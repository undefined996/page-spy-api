package route

import (
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"strconv"

	"github.com/HuolalaTech/page-spy-api/config"
	"github.com/HuolalaTech/page-spy-api/serve/common"
	selfMiddleware "github.com/HuolalaTech/page-spy-api/serve/middleware"
	"github.com/HuolalaTech/page-spy-api/serve/socket"
	"github.com/HuolalaTech/page-spy-api/static"
	"github.com/HuolalaTech/page-spy-api/storage"
	"github.com/labstack/echo/v4"
)

func NewEcho(socket *socket.WebSocket, core *CoreApi, config *config.Config, staticConfig *config.StaticConfig) *echo.Echo {
	e := echo.New()
	e.Use(selfMiddleware.Logger())
	e.Use(selfMiddleware.Error())
	e.Use(selfMiddleware.CORS(config))
	e.HidePort = true
	e.HideBanner = true
	route := e.Group("/api/v1")
	route.GET("/room/list", func(c echo.Context) error {
		socket.ListRooms(c.Response(), c.Request())
		return nil
	})

	route.POST("/room/create", func(c echo.Context) error {
		socket.CreateRoom(c.Response(), c.Request())
		return nil
	})

	route.GET("/ws/room/join", func(c echo.Context) error {
		socket.JoinRoom(c.Response(), c.Request())
		return nil
	})

	route.GET("/log/download", func(c echo.Context) error {
		return nil
	})

	route.GET("/local/log/download", func(c echo.Context) error {
		fileId := c.QueryParam("fileId")

		file, err := core.GetFile(fileId)
		if err != nil {
			return err
		}

		defer file.File.Close()
		c.Response().Header().Set("Content-Disposition", "attachment; filename="+file.Name)
		c.Response().Header().Set("Content-Type", "application/octet-stream")
		c.Response().Header().Set("Content-Length", strconv.FormatInt(file.Size, 10))

		_, err = io.Copy(c.Response().Writer, file.File)
		if err != nil {
			return err
		}

		return nil
	})

	route.DELETE("/local/log/delete", func(c echo.Context) error {
		fileId := c.QueryParam("fileId")

		err := core.DeleteFile(fileId)
		if err != nil {
			return err
		}

		return c.JSON(200, common.NewSuccessResponse(true))
	})

	route.POST("/log/upload", func(c echo.Context) error {
		file, err := c.FormFile("log")
		if err != nil {
			return err
		}

		src, err := file.Open()
		if err != nil {
			return fmt.Errorf("open upload file error: %w", err)
		}

		defer src.Close()
		logFile := &storage.LogFile{
			Name: file.Filename,
			Size: file.Size,
			File: src,
		}

		createFile, err := core.CreateFile(logFile)
		if err != nil {
			return err
		}

		return c.JSON(200, common.NewSuccessResponse(createFile))
	})

	if staticConfig != nil {
		dist, err := fs.Sub(staticConfig.Files, "dist")
		if err != nil {
			panic(err)
		}

		ff := static.NewFallbackFS(
			dist,
			"index.html",
		)

		e.GET(
			"/*",
			echo.WrapHandler(
				http.FileServer(http.FS(ff))),
			selfMiddleware.Cache(),
		)
	}

	return e
}
