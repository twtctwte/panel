package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/gookit/color"
	"github.com/goravel/framework/contracts/console"
	"github.com/goravel/framework/contracts/console/command"
	"github.com/goravel/framework/facades"
	"github.com/goravel/framework/support/carbon"
	"github.com/spf13/cast"

	"panel/app/models"
	"panel/internal"
	"panel/internal/services"
	"panel/pkg/tools"
)

// Panel 面板命令行
type Panel struct {
}

// Signature The name and signature of the console command.
func (receiver *Panel) Signature() string {
	return "panel"
}

// Description The console command description.
func (receiver *Panel) Description() string {
	ctx := context.Background()
	return facades.Lang(ctx).Get("commands.panel.description")
}

// Extend The console command extend.
func (receiver *Panel) Extend() command.Extend {
	return command.Extend{
		Category: "panel",
	}
}

// Handle Execute the console command.
func (receiver *Panel) Handle(ctx console.Context) error {
	action := ctx.Argument(0)
	arg1 := ctx.Argument(1)
	arg2 := ctx.Argument(2)
	arg3 := ctx.Argument(3)
	arg4 := ctx.Argument(4)
	arg5 := ctx.Argument(5)

	translate := facades.Lang(context.Background())

	switch action {
	case "init":
		var check models.User
		err := facades.Orm().Query().FirstOrFail(&check)
		if err == nil {
			color.Redln(translate.Get("commands.panel.init.exist"))
			return nil
		}

		settings := []models.Setting{{Key: models.SettingKeyName, Value: "耗子 Linux 面板"}, {Key: models.SettingKeyMonitor, Value: "1"}, {Key: models.SettingKeyMonitorDays, Value: "30"}, {Key: models.SettingKeyBackupPath, Value: "/www/backup"}, {Key: models.SettingKeyWebsitePath, Value: "/www/wwwroot"}, {Key: models.SettingKeyVersion, Value: facades.Config().GetString("panel.version")}}
		err = facades.Orm().Query().Create(&settings)
		if err != nil {
			color.Redln(translate.Get("commands.panel.init.fail"))
			return nil
		}

		hash, err := facades.Hash().Make(tools.RandomString(32))
		if err != nil {
			color.Redln(translate.Get("commands.panel.init.fail"))
			return nil
		}

		user := services.NewUserImpl()
		_, err = user.Create("admin", hash)
		if err != nil {
			color.Redln(translate.Get("commands.panel.init.adminFail"))
			return nil
		}

		color.Greenln(translate.Get("commands.panel.init.success"))

	case "update":
		var task models.Task
		err := facades.Orm().Query().Where("status", models.TaskStatusRunning).OrWhere("status", models.TaskStatusWaiting).FirstOrFail(&task)
		if err == nil {
			color.Redln(translate.Get("commands.panel.update.taskCheck"))
			return nil
		}

		panel, err := tools.GetLatestPanelVersion()
		if err != nil {
			color.Redln(translate.Get("commands.panel.update.versionFail"))
			return err
		}

		internal.Status = internal.StatusUpgrade
		if err = tools.UpdatePanel(panel); err != nil {
			internal.Status = internal.StatusFailed
			color.Redln(translate.Get("commands.panel.update.fail") + ": " + err.Error())
			return nil
		}

		internal.Status = internal.StatusNormal
		color.Greenln(translate.Get("commands.panel.update.success"))
		tools.RestartPanel()

	case "getInfo":
		var user models.User
		err := facades.Orm().Query().Where("id", 1).FirstOrFail(&user)
		if err != nil {
			color.Redln(translate.Get("commands.panel.getInfo.adminGetFail"))
			return nil
		}

		password := tools.RandomString(16)
		hash, err := facades.Hash().Make(password)
		if err != nil {
			color.Redln(translate.Get("commands.panel.getInfo.passwordGenerationFail"))
			return nil
		}
		user.Username = tools.RandomString(8)
		user.Password = hash
		if user.Email == "" {
			user.Email = tools.RandomString(8) + "@example.com"
		}

		err = facades.Orm().Query().Save(&user)
		if err != nil {
			color.Redln(translate.Get("commands.panel.getInfo.adminSaveFail"))
			return nil
		}

		port, err := tools.Exec(`cat /www/panel/panel.conf | grep APP_PORT | awk -F '=' '{print $2}' | tr -d '\n'`)
		if err != nil {
			color.Redln(translate.Get("commands.panel.portFail"))
			return nil
		}
		ip, err := tools.GetPublicIP()
		if err != nil {
			ip = "127.0.0.1"
		}
		protocol := "http"
		if facades.Config().GetBool("panel.ssl") {
			protocol = "https"
		}

		color.Greenln(translate.Get("commands.panel.getInfo.username") + ": " + user.Username)
		color.Greenln(translate.Get("commands.panel.getInfo.password") + ": " + password)
		color.Greenln(translate.Get("commands.panel.port") + ": " + port)
		color.Greenln(translate.Get("commands.panel.entrance") + ": " + facades.Config().GetString("http.entrance"))
		color.Greenln(translate.Get("commands.panel.getInfo.address") + ": " + protocol + "://" + ip + ":" + port + facades.Config().GetString("http.entrance"))

	case "getPort":
		port, err := tools.Exec(`cat /www/panel/panel.conf | grep APP_PORT | awk -F '=' '{print $2}' | tr -d '\n'`)
		if err != nil {
			color.Redln(translate.Get("commands.panel.portFail"))
			return nil
		}

		color.Greenln(translate.Get("commands.panel.port") + ": " + port)

	case "getEntrance":
		color.Greenln(translate.Get("commands.panel.entrance") + ": " + facades.Config().GetString("http.entrance"))

	case "deleteEntrance":
		oldEntrance, err := tools.Exec(`cat /www/panel/panel.conf | grep APP_ENTRANCE | awk -F '=' '{print $2}' | tr -d '\n'`)
		if err != nil {
			color.Redln(translate.Get("commands.panel.deleteEntrance.fail"))
			return nil
		}
		if _, err = tools.Exec("sed -i 's!APP_ENTRANCE=" + oldEntrance + "!APP_ENTRANCE=/!g' /www/panel/panel.conf"); err != nil {
			color.Redln(translate.Get("commands.panel.deleteEntrance.fail"))
			return nil
		}

		color.Greenln(translate.Get("commands.panel.deleteEntrance.success"))

	case "writePlugin":
		slug := arg1
		version := arg2
		if len(slug) == 0 || len(version) == 0 {
			color.Redln(translate.Get("commands.panel.writePlugin.paramFail"))
			return nil
		}

		var plugin models.Plugin
		err := facades.Orm().Query().UpdateOrCreate(&plugin, models.Plugin{
			Slug: slug,
		}, models.Plugin{
			Version: version,
		})

		if err != nil {
			color.Redln(translate.Get("commands.panel.writePlugin.fail"))
			return nil
		}

		color.Greenln(translate.Get("commands.panel.writePlugin.success"))

	case "deletePlugin":
		slug := arg1
		if len(slug) == 0 {
			color.Redln(translate.Get("commands.panel.deletePlugin.paramFail"))
			return nil
		}

		_, err := facades.Orm().Query().Where("slug", slug).Delete(&models.Plugin{})
		if err != nil {
			color.Redln(translate.Get("commands.panel.deletePlugin.fail"))
			return nil
		}

		color.Greenln(translate.Get("commands.panel.deletePlugin.success"))

	case "writeMysqlPassword":
		password := arg1
		if len(password) == 0 {
			color.Redln(translate.Get("commands.panel.writeMysqlPassword.paramFail"))
			return nil
		}

		var setting models.Setting
		err := facades.Orm().Query().UpdateOrCreate(&setting, models.Setting{
			Key: models.SettingKeyMysqlRootPassword,
		}, models.Setting{
			Value: password,
		})

		if err != nil {
			color.Redln(translate.Get("commands.panel.writeMysqlPassword.fail"))
			return nil
		}

		color.Greenln(translate.Get("commands.panel.writeMysqlPassword.success"))

	case "cleanTask":
		_, err := facades.Orm().Query().Model(&models.Task{}).Where("status", models.TaskStatusRunning).OrWhere("status", models.TaskStatusWaiting).Update("status", models.TaskStatusFailed)
		if err != nil {
			color.Redln(translate.Get("commands.panel.cleanTask.fail"))
			return nil
		}

		color.Greenln(translate.Get("commands.panel.cleanTask.success"))

	case "backup":
		backupType := arg1
		name := arg2
		path := arg3
		save := arg4
		hr := `+----------------------------------------------------`
		if len(backupType) == 0 || len(name) == 0 || len(path) == 0 || len(save) == 0 {
			color.Redln(translate.Get("commands.panel.backup.paramFail"))
			return nil
		}

		color.Greenln(hr)
		color.Greenln("★ " + translate.Get("commands.panel.backup.start") + " [" + carbon.Now().ToDateTimeString() + "]")
		color.Greenln(hr)

		if !tools.Exists(path) {
			if err := tools.Mkdir(path, 0644); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.backupDirFail") + ": " + err.Error())
				return nil
			}
		}

		switch backupType {
		case "website":
			color.Yellowln("|-" + translate.Get("commands.panel.backup.targetSite") + ": " + name)
			var website models.Website
			if err := facades.Orm().Query().Where("name", name).FirstOrFail(&website); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.siteNotExist"))
				color.Greenln(hr)
				return nil
			}

			backupFile := path + "/" + website.Name + "_" + carbon.Now().ToShortDateTimeString() + ".zip"
			if _, err := tools.Exec(`cd '` + website.Path + `' && zip -r '` + backupFile + `' .`); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.backupFail") + ": " + err.Error())
				return nil
			}
			color.Greenln("|-" + translate.Get("commands.panel.backup.backupSuccess"))

		case "mysql":
			rootPassword := services.NewSettingImpl().Get(models.SettingKeyMysqlRootPassword)
			backupFile := name + "_" + carbon.Now().ToShortDateTimeString() + ".sql"

			err := os.Setenv("MYSQL_PWD", rootPassword)
			if err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.mysqlBackupFail") + ": " + err.Error())
				color.Greenln(hr)
				return nil
			}

			color.Greenln("|-" + translate.Get("commands.panel.backup.targetMysql") + ": " + name)
			color.Greenln("|-" + translate.Get("commands.panel.backup.startExport"))
			if _, err = tools.Exec(`mysqldump -uroot ` + name + ` > /tmp/` + backupFile + ` 2>&1`); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.exportFail") + ": " + err.Error())
				return nil
			}
			color.Greenln("|-" + translate.Get("commands.panel.backup.exportSuccess"))
			color.Greenln("|-" + translate.Get("commands.panel.backup.startCompress"))
			if _, err = tools.Exec("cd /tmp && zip -r " + backupFile + ".zip " + backupFile); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.compressFail") + ": " + err.Error())
				return nil
			}
			if err := tools.Remove("/tmp/" + backupFile); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.deleteFail") + ": " + err.Error())
				return nil
			}
			color.Greenln("|-" + translate.Get("commands.panel.backup.compressSuccess"))
			color.Greenln("|-" + translate.Get("commands.panel.backup.startMove"))
			if err := tools.Mv("/tmp/"+backupFile+".zip", path+"/"+backupFile+".zip"); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.moveFail") + ": " + err.Error())
				return nil
			}
			color.Greenln("|-" + translate.Get("commands.panel.backup.moveSuccess"))
			_ = os.Unsetenv("MYSQL_PWD")
			color.Greenln("|-" + translate.Get("commands.panel.backup.success"))

		case "postgresql":
			backupFile := name + "_" + carbon.Now().ToShortDateTimeString() + ".sql"
			check, err := tools.Exec(`su - postgres -c "psql -l" 2>&1`)
			if err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.databaseGetFail") + ": " + err.Error())
				color.Greenln(hr)
				return nil
			}
			if !strings.Contains(check, name) {
				color.Redln("|-" + translate.Get("commands.panel.backup.databaseNotExist"))
				color.Greenln(hr)
				return nil
			}

			color.Greenln("|-" + translate.Get("commands.panel.backup.targetPostgres") + ": " + name)
			color.Greenln("|-" + translate.Get("commands.panel.backup.startExport"))
			if _, err = tools.Exec(`su - postgres -c "pg_dump '` + name + `'" > /tmp/` + backupFile + ` 2>&1`); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.exportFail") + ": " + err.Error())
				return nil
			}
			color.Greenln("|-" + translate.Get("commands.panel.backup.exportSuccess"))
			color.Greenln("|-" + translate.Get("commands.panel.backup.startCompress"))
			if _, err = tools.Exec("cd /tmp && zip -r " + backupFile + ".zip " + backupFile); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.compressFail") + ": " + err.Error())
				return nil
			}
			if err := tools.Remove("/tmp/" + backupFile); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.deleteFail") + ": " + err.Error())
				return nil
			}
			color.Greenln("|-" + translate.Get("commands.panel.backup.compressSuccess"))
			color.Greenln("|-" + translate.Get("commands.panel.backup.startMove"))
			if err := tools.Mv("/tmp/"+backupFile+".zip", path+"/"+backupFile+".zip"); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.moveFail") + ": " + err.Error())
				return nil
			}
			color.Greenln("|-" + translate.Get("commands.panel.backup.moveSuccess"))
			color.Greenln("|-" + translate.Get("commands.panel.backup.success"))
		}

		color.Greenln(hr)
		files, err := os.ReadDir(path)
		if err != nil {
			color.Redln("|-" + translate.Get("commands.panel.backup.cleanupFail") + ": " + err.Error())
			return nil
		}
		var filteredFiles []os.FileInfo
		for _, file := range files {
			if strings.HasPrefix(file.Name(), name) && strings.HasSuffix(file.Name(), ".zip") {
				fileInfo, err := os.Stat(filepath.Join(path, file.Name()))
				if err != nil {
					continue
				}
				filteredFiles = append(filteredFiles, fileInfo)
			}
		}
		sort.Slice(filteredFiles, func(i, j int) bool {
			return filteredFiles[i].ModTime().After(filteredFiles[j].ModTime())
		})
		for i := cast.ToInt(save); i < len(filteredFiles); i++ {
			fileToDelete := filepath.Join(path, filteredFiles[i].Name())
			color.Yellowln("|-" + translate.Get("commands.panel.backup.cleanBackup") + ": " + fileToDelete)
			if err := tools.Remove(fileToDelete); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.backup.cleanupFail") + ": " + err.Error())
				return nil
			}
		}
		color.Greenln("|-" + translate.Get("commands.panel.backup.cleanupSuccess"))
		color.Greenln(hr)
		color.Greenln("☆ " + translate.Get("commands.panel.backup.success") + " [" + carbon.Now().ToDateTimeString() + "]")
		color.Greenln(hr)

	case "cutoff":
		name := arg1
		save := arg2
		hr := `+----------------------------------------------------`
		if len(name) == 0 || len(save) == 0 {
			color.Redln(translate.Get("commands.panel.cutoff.paramFail"))
			return nil
		}

		color.Greenln(hr)
		color.Greenln("★ " + translate.Get("commands.panel.cutoff.start") + " [" + carbon.Now().ToDateTimeString() + "]")
		color.Greenln(hr)

		color.Yellowln("|-" + translate.Get("commands.panel.cutoff.targetSite") + ": " + name)
		var website models.Website
		if err := facades.Orm().Query().Where("name", name).FirstOrFail(&website); err != nil {
			color.Redln("|-" + translate.Get("commands.panel.cutoff.siteNotExist"))
			color.Greenln(hr)
			return nil
		}

		logPath := "/www/wwwlogs/" + website.Name + ".log"
		if !tools.Exists(logPath) {
			color.Redln("|-" + translate.Get("commands.panel.cutoff.logNotExist"))
			color.Greenln(hr)
			return nil
		}

		backupPath := "/www/wwwlogs/" + website.Name + "_" + carbon.Now().ToShortDateTimeString() + ".log.zip"
		if _, err := tools.Exec(`cd /www/wwwlogs && zip -r ` + backupPath + ` ` + website.Name + ".log"); err != nil {
			color.Redln("|-" + translate.Get("commands.panel.cutoff.backupFail") + ": " + err.Error())
			return nil
		}
		if _, err := tools.Exec(`echo "" > ` + logPath); err != nil {
			color.Redln("|-" + translate.Get("commands.panel.cutoff.clearFail") + ": " + err.Error())
			return nil
		}
		color.Greenln("|-" + translate.Get("commands.panel.cutoff.cutSuccess"))

		color.Greenln(hr)
		files, err := os.ReadDir("/www/wwwlogs")
		if err != nil {
			color.Redln("|-" + translate.Get("commands.panel.cutoff.cleanupFail") + ": " + err.Error())
			return nil
		}
		var filteredFiles []os.FileInfo
		for _, file := range files {
			if strings.HasPrefix(file.Name(), website.Name) && strings.HasSuffix(file.Name(), ".log.zip") {
				fileInfo, err := os.Stat(filepath.Join("/www/wwwlogs", file.Name()))
				if err != nil {
					continue
				}
				filteredFiles = append(filteredFiles, fileInfo)
			}
		}
		sort.Slice(filteredFiles, func(i, j int) bool {
			return filteredFiles[i].ModTime().After(filteredFiles[j].ModTime())
		})
		for i := cast.ToInt(save); i < len(filteredFiles); i++ {
			fileToDelete := filepath.Join("/www/wwwlogs", filteredFiles[i].Name())
			color.Yellowln("|-" + translate.Get("commands.panel.cutoff.clearLog") + ": " + fileToDelete)
			if err := tools.Remove(fileToDelete); err != nil {
				color.Redln("|-" + translate.Get("commands.panel.cutoff.cleanupFail") + ": " + err.Error())
				return nil
			}
		}
		color.Greenln("|-" + translate.Get("commands.panel.cutoff.cleanupSuccess"))
		color.Greenln(hr)
		color.Greenln("☆ " + translate.Get("commands.panel.cutoff.end") + " [" + carbon.Now().ToDateTimeString() + "]")
		color.Greenln(hr)

	case "writeSite":
		name := arg1
		status := cast.ToBool(arg2)
		path := arg3
		php := cast.ToInt(arg4)
		ssl := cast.ToBool(ctx.Argument(5))
		if len(name) == 0 || len(path) == 0 {
			color.Redln(translate.Get("commands.panel.writeSite.paramFail"))
			return nil
		}

		var website models.Website
		if err := facades.Orm().Query().Where("name", name).FirstOrFail(&website); err == nil {
			color.Redln(translate.Get("commands.panel.writeSite.siteExist"))
			return nil
		}

		_, err := os.Stat(path)
		if os.IsNotExist(err) {
			color.Redln(translate.Get("commands.panel.writeSite.pathNotExist"))
			return nil
		}

		err = facades.Orm().Query().Create(&models.Website{
			Name:   name,
			Status: status,
			Path:   path,
			Php:    php,
			Ssl:    ssl,
		})
		if err != nil {
			color.Redln(translate.Get("commands.panel.writeSite.fail"))
			return nil
		}

		color.Greenln(translate.Get("commands.panel.writeSite.success"))

	case "deleteSite":
		name := arg1
		if len(name) == 0 {
			color.Redln(translate.Get("commands.panel.deleteSite.paramFail"))
			return nil
		}

		_, err := facades.Orm().Query().Where("name", name).Delete(&models.Website{})
		if err != nil {
			color.Redln(translate.Get("commands.panel.deleteSite.fail"))
			return nil
		}

		color.Greenln(translate.Get("commands.panel.deleteSite.success"))

	case "writeSetting":
		key := arg1
		value := arg2
		if len(key) == 0 || len(value) == 0 {
			color.Redln(translate.Get("commands.panel.writeSetting.paramFail"))
			return nil
		}

		var setting models.Setting
		err := facades.Orm().Query().UpdateOrCreate(&setting, models.Setting{
			Key: key,
		}, models.Setting{
			Value: value,
		})
		if err != nil {
			color.Redln(translate.Get("commands.panel.writeSetting.fail"))
			return nil
		}

		color.Greenln(translate.Get("commands.panel.writeSetting.success"))

	case "getSetting":
		key := arg1
		if len(key) == 0 {
			color.Redln(translate.Get("commands.panel.getSetting.paramFail"))
			return nil
		}

		var setting models.Setting
		if err := facades.Orm().Query().Where("key", key).FirstOrFail(&setting); err != nil {
			return nil
		}

		fmt.Printf("%s", setting.Value)

	case "deleteSetting":
		key := arg1
		if len(key) == 0 {
			color.Redln(translate.Get("commands.panel.deleteSetting.paramFail"))
			return nil
		}

		_, err := facades.Orm().Query().Where("key", key).Delete(&models.Setting{})
		if err != nil {
			color.Redln(translate.Get("commands.panel.deleteSetting.fail"))
			return nil
		}

		color.Greenln(translate.Get("commands.panel.deleteSetting.success"))

	case "addSite":
		name := arg1
		domain := arg2
		port := arg3
		path := arg4
		php := arg5
		if len(name) == 0 || len(domain) == 0 || len(port) == 0 || len(path) == 0 {
			color.Redln(translate.Get("commands.panel.addSite.paramFail"))
			return nil
		}

		domains := strings.Split(domain, ",")
		ports := strings.Split(port, ",")
		if len(domains) == 0 || len(ports) == 0 {
			color.Redln(translate.Get("commands.panel.addSite.paramFail"))
			return nil
		}

		var uintPorts []uint
		for _, p := range ports {
			uintPorts = append(uintPorts, cast.ToUint(p))
		}

		website := services.NewWebsiteImpl()
		id, err := website.GetIDByName(name)
		if err != nil {
			color.Redln(err.Error())
			return nil
		}
		if id != 0 {
			color.Redln(translate.Get("commands.panel.addSite.siteExist"))
			return nil
		}

		_, err = website.Add(internal.PanelWebsite{
			Name:    name,
			Status:  true,
			Domains: domains,
			Ports:   uintPorts,
			Path:    path,
			Php:     php,
			Ssl:     false,
			Db:      false,
		})
		if err != nil {
			color.Redln(err.Error())
			return nil
		}

		color.Greenln(translate.Get("commands.panel.addSite.success"))

	case "removeSite":
		name := arg1
		if len(name) == 0 {
			color.Redln(translate.Get("commands.panel.removeSite.paramFail"))
			return nil
		}

		website := services.NewWebsiteImpl()
		id, err := website.GetIDByName(name)
		if err != nil {
			color.Redln(err.Error())
			return nil
		}
		if id == 0 {
			color.Redln(translate.Get("commands.panel.removeSite.siteNotExist"))
			return nil
		}

		if err = website.Delete(id); err != nil {
			color.Redln(err.Error())
			return nil
		}

		color.Greenln(translate.Get("commands.panel.removeSite.success"))

	case "installPlugin":
		slug := arg1
		if len(slug) == 0 {
			color.Redln(translate.Get("commands.panel.installPlugin.paramFail"))
			return nil
		}

		plugin := services.NewPluginImpl()
		if err := plugin.Install(slug); err != nil {
			color.Redln(err.Error())
			return nil
		}

		color.Greenln(translate.Get("commands.panel.installPlugin.success"))

	case "uninstallPlugin":
		slug := arg1
		if len(slug) == 0 {
			color.Redln(translate.Get("commands.panel.uninstallPlugin.paramFail"))
			return nil
		}

		plugin := services.NewPluginImpl()
		if err := plugin.Uninstall(slug); err != nil {
			color.Redln(err.Error())
			return nil
		}

		color.Greenln(translate.Get("commands.panel.uninstallPlugin.success"))

	case "updatePlugin":
		slug := arg1
		if len(slug) == 0 {
			color.Redln(translate.Get("commands.panel.updatePlugin.paramFail"))
			return nil
		}

		plugin := services.NewPluginImpl()
		if err := plugin.Update(slug); err != nil {
			color.Redln(err.Error())
			return nil
		}

		color.Greenln(translate.Get("commands.panel.updatePlugin.success"))

	default:
		color.Yellowln(facades.Config().GetString("panel.name") + " - " + translate.Get("commands.panel.tool") + " - " + facades.Config().GetString("panel.version"))
		color.Greenln(translate.Get("commands.panel.use") + "：")
		color.Greenln("panel update " + translate.Get("commands.panel.update.description"))
		color.Greenln("panel getInfo " + translate.Get("commands.panel.getInfo.description"))
		color.Greenln("panel getPort " + translate.Get("commands.panel.getPort.description"))
		color.Greenln("panel getEntrance " + translate.Get("commands.panel.getEntrance.description"))
		color.Greenln("panel deleteEntrance " + translate.Get("commands.panel.deleteEntrance.description"))
		color.Greenln("panel cleanTask " + translate.Get("commands.panel.cleanTask.description"))
		color.Greenln("panel backup {website/mysql/postgresql} {name} {path} {save_copies} " + translate.Get("commands.panel.backup.description"))
		color.Greenln("panel cutoff {website_name} {save_copies} " + translate.Get("commands.panel.cutoff.description"))
		color.Greenln("panel installPlugin {slug} " + translate.Get("commands.panel.installPlugin.description"))
		color.Greenln("panel uninstallPlugin {slug} " + translate.Get("commands.panel.uninstallPlugin.description"))
		color.Greenln("panel updatePlugin {slug} " + translate.Get("commands.panel.updatePlugin.description"))
		color.Greenln("panel addSite {name} {domain} {port} {path} {php} " + translate.Get("commands.panel.addSite.description"))
		color.Greenln("panel removeSite {name} " + translate.Get("commands.panel.removeSite.description"))
		color.Redln(translate.Get("commands.panel.forDeveloper") + ":")
		color.Yellowln("panel init " + translate.Get("commands.panel.init.description"))
		color.Yellowln("panel writePlugin {slug} {version} " + translate.Get("commands.panel.writePlugin.description"))
		color.Yellowln("panel deletePlugin {slug} " + translate.Get("commands.panel.deletePlugin.description"))
		color.Yellowln("panel writeMysqlPassword {password} " + translate.Get("commands.panel.writeMysqlPassword.description"))
		color.Yellowln("panel writeSite {name} {status} {path} {php} {ssl} " + translate.Get("commands.panel.writeSite.description"))
		color.Yellowln("panel deleteSite {name} " + translate.Get("commands.panel.deleteSite.description"))
		color.Yellowln("panel getSetting {name} " + translate.Get("commands.panel.getSetting.description"))
		color.Yellowln("panel writeSetting {name} {value} " + translate.Get("commands.panel.writeSetting.description"))
		color.Yellowln("panel deleteSetting {name} " + translate.Get("commands.panel.deleteSetting.description"))
	}

	return nil
}
