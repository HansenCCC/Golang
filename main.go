package main

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	_ "github.com/go-sql-driver/mysql"

	"github.com/gin-gonic/gin"

	"github.com/gin-contrib/cors"
)

var databases *sql.DB

// ———————————————————————————————— MAIN ————————————————————————————————

func main() {
	databases = ConnectMySql()
	if databases != nil {
		defer databases.Close()

		RequestNetwork()
	}
}

// ———————————————————————————————— ROUTER ————————————————————————————————

// 监听网络请求
func RequestNetwork() {
	r := gin.Default()

	// 添加Cors()中间件 -> 解决跨域问题
	r.Use(cors.Default())

	r.GET("/", func(ctx *gin.Context) {
		clientIP := ctx.ClientIP()
		ctx.JSON(200, gin.H{
			"clientIP": clientIP,
		})
	})
	r.GET("/game/ranking", func(ctx *gin.Context) {
		moveDataList := GetGameRanking(true)
		timeDataList := GetGameRanking(false)
		ctx.JSON(200, gin.H{
			"msg":          "success",
			"moveDataList": moveDataList,
			"timeDataList": timeDataList,
		})
	})
	r.POST("/game/adddata", func(ctx *gin.Context) {
		var gameData GameRanking
		if err := ctx.ShouldBind(&gameData); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"msg":   "invalid request body",
				"error": err,
			})
			return
		}
		AddGameData(gameData)
		ctx.JSON(200, gin.H{
			"msg": "success",
		})
	})
	r.POST("/game/init", func(ctx *gin.Context) {
		var gamerInfo GamerUserInfo
		if err := ctx.ShouldBind(&gamerInfo); err != nil {
			ctx.JSON(http.StatusBadRequest, gin.H{
				"msg":   "invalid request body",
				"error": err.Error(),
			})
			return
		}
		gamerInfo.IP = ctx.ClientIP()
		sqlErr := AddGamerData(gamerInfo)
		errorMessage := ""
		if sqlErr != nil {
			errorMessage = sqlErr.Error()
		}
		ctx.JSON(200, gin.H{
			"msg":   "success",
			"error": errorMessage,
		})
	})
	r.Run(":8081")
}

// ———————————————————————————————— MODEL ————————————————————————————————

type GameRanking struct {
	Id          int    `json:"id"`
	Name        string `json:"name"`
	MoveCount   int    `json:"moveCount"`
	CreatedTime string `json:"createdTime"`
	OverTime    string `json:"overTime"`
	Duration    int    `json:"duration"`
	Udid        string `json:"udid"`
}

type GamerUserInfo struct {
	Id        int    `json:"id"`
	Name      string `json:"name"`
	PlayCount int    `json:"play_count"`
	Udid      string `json:"udid"`
	IP        string `json:"ip"`
}

// ———————————————————————————————— API ————————————————————————————————
// 获取游戏排名
func GetGameRanking(isMoveType bool) []GameRanking {
	// 根据时间排序，获取用时最短的用户列表
	sqlStr := "select id, gamer_username, gamer_move_count, created_at, gameover_time, TIMESTAMPDIFF(SECOND, created_at, gameover_time) AS duration from yp_gamer_data ORDER BY duration ASC LIMIT 20"
	if isMoveType {
		sqlStr = "select id, gamer_username, gamer_move_count, created_at, gameover_time, TIMESTAMPDIFF(SECOND, created_at, gameover_time) AS duration from yp_gamer_data ORDER BY gamer_move_count ASC LIMIT 20"
	}
	dataList, _ := databases.Query(sqlStr)
	defer dataList.Close()

	gameInfos := []GameRanking{}
	for dataList.Next() {
		var gameInfo GameRanking
		if err := dataList.Scan(&gameInfo.Id, &gameInfo.Name, &gameInfo.MoveCount, &gameInfo.CreatedTime, &gameInfo.OverTime, &gameInfo.Duration); err != nil {
			fmt.Println("Failed to scan row: ", err)
			return gameInfos
		}
		gameInfos = append(gameInfos, gameInfo)
	}
	return gameInfos
}

// 增加游戏数据
func AddGameData(networkGameData GameRanking) {
	var gameData GameRanking = networkGameData
	fmt.Println(gameData.Name)
	fmt.Println(gameData.OverTime)
	fmt.Println(gameData.MoveCount)
	fmt.Println(gameData.CreatedTime)
	fmt.Println(gameData.Udid)
	if (gameData.OverTime > gameData.CreatedTime) && len(gameData.OverTime) > 0 && len(gameData.CreatedTime) > 0 && len(gameData.Udid) == 32 {
		// 满足添加条件
		sqlStr := "INSERT INTO yp_gamer_data(gamer_username, gamer_move_count, created_at, gameover_time, uuid) VALUES(?, ?, ?, ?, ?)"
		result, err := databases.Exec(sqlStr, gameData.Name, gameData.MoveCount, gameData.CreatedTime, gameData.OverTime, gameData.Udid)
		fmt.Println(result)
		fmt.Println(err)
	}
}

// 增加用户
func AddGamerData(gamer GamerUserInfo) error {
	if len(gamer.Udid) != 32 {
		return errors.New("unknown error")
	}
	var count int
	err := databases.QueryRow("SELECT COUNT(*) FROM yp_gamers WHERE uuid = ?", gamer.Udid).Scan(&count)
	if err != nil {
		return err
	}
	if count > 0 {
		currentTime := time.Now().Format("2006-01-02 15:04:05.000")
		_, err = databases.Exec("UPDATE yp_gamers SET play_count = play_count + 1,updated_at = ?  WHERE uuid = ?", currentTime, gamer.Udid)
		if err != nil {
			return err
		}
	} else {
		currentTime := time.Now().Format("2006-01-02 15:04:05.000")
		_, err = databases.Exec("INSERT INTO yp_gamers (name, play_count, uuid, ip, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)", gamer.Name, gamer.PlayCount, gamer.Udid, gamer.IP, currentTime, currentTime)
		if err != nil {
			return err
		}
	}
	return nil
}

// 数据初始化
func ConnectMySql() (db *sql.DB) {
	// fmt.Sprintf("%s:%s@tcp(%s:%s)/", i.UserName, i.Password, i.Host, i.Port)
	dsn := "root:ChengHengSheng1995!@tcp(121.43.188.78:3306)/chenghengsheng"
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		fmt.Println("🚀🚀🚀 数据库 Open() 失败：", err)
		return nil
	}
	// 尝试去连接
	err2 := db.Ping()
	if err2 != nil {
		fmt.Println("🚀🚀🚀 数据库 Ping() 失败：", err2)
		return nil
	}
	return db
}
