// Package services 插件服务
package services

import (
	"errors"

	"github.com/goravel/framework/facades"

	"panel/app/models"
	"panel/internal"
)

type PluginImpl struct {
	task internal.Task
}

func NewPluginImpl() *PluginImpl {
	return &PluginImpl{
		task: NewTaskImpl(),
	}
}

// AllInstalled 获取已安装的所有插件
func (r *PluginImpl) AllInstalled() ([]models.Plugin, error) {
	var plugins []models.Plugin
	if err := facades.Orm().Query().Get(&plugins); err != nil {
		return plugins, err
	}

	return plugins, nil
}

// All 获取所有插件
func (r *PluginImpl) All() []internal.PanelPlugin {
	var plugins = []internal.PanelPlugin{
		internal.PluginOpenResty,
		internal.PluginMySQL57,
		internal.PluginMySQL80,
		internal.PluginMySQL84,
		internal.PluginPostgreSQL15,
		internal.PluginPostgreSQL16,
		internal.PluginPHP74,
		internal.PluginPHP80,
		internal.PluginPHP81,
		internal.PluginPHP82,
		internal.PluginPHP83,
		internal.PluginPHPMyAdmin,
		internal.PluginPureFTPd,
		internal.PluginRedis,
		internal.PluginS3fs,
		internal.PluginRsync,
		internal.PluginSupervisor,
		internal.PluginFail2ban,
		internal.PluginToolBox,
	}

	return plugins
}

// GetBySlug 根据slug获取插件
func (r *PluginImpl) GetBySlug(slug string) internal.PanelPlugin {
	for _, item := range r.All() {
		if item.Slug == slug {
			return item
		}
	}

	return internal.PanelPlugin{}
}

// GetInstalledBySlug 根据slug获取已安装的插件
func (r *PluginImpl) GetInstalledBySlug(slug string) models.Plugin {
	var plugin models.Plugin
	if err := facades.Orm().Query().Where("slug", slug).Get(&plugin); err != nil {
		return plugin
	}

	return plugin
}

// Install 安装插件
func (r *PluginImpl) Install(slug string) error {
	plugin := r.GetBySlug(slug)
	installedPlugin := r.GetInstalledBySlug(slug)
	installedPlugins, err := r.AllInstalled()
	if err != nil {
		return err
	}

	if installedPlugin.ID != 0 {
		return errors.New("插件已安装")
	}

	pluginsMap := make(map[string]bool)

	for _, p := range installedPlugins {
		pluginsMap[p.Slug] = true
	}

	for _, require := range plugin.Requires {
		_, requireFound := pluginsMap[require]
		if !requireFound {
			return errors.New("插件 " + slug + " 需要依赖 " + require + " 插件")
		}
	}

	for _, exclude := range plugin.Excludes {
		_, excludeFound := pluginsMap[exclude]
		if excludeFound {
			return errors.New("插件 " + slug + " 不兼容 " + exclude + " 插件")
		}
	}

	var task models.Task
	task.Name = "安装插件 " + plugin.Name
	task.Status = models.TaskStatusWaiting
	task.Shell = plugin.Install + ` >> '/tmp/` + plugin.Slug + `.log' 2>&1`
	task.Log = "/tmp/" + plugin.Slug + ".log"
	if err = facades.Orm().Query().Create(&task); err != nil {
		return errors.New("创建任务失败")
	}

	r.task.Process(task.ID)
	return nil
}

// Uninstall 卸载插件
func (r *PluginImpl) Uninstall(slug string) error {
	plugin := r.GetBySlug(slug)
	installedPlugin := r.GetInstalledBySlug(slug)
	installedPlugins, err := r.AllInstalled()
	if err != nil {
		return err
	}

	if installedPlugin.ID == 0 {
		return errors.New("插件未安装")
	}

	pluginsMap := make(map[string]bool)

	for _, p := range installedPlugins {
		pluginsMap[p.Slug] = true
	}

	for _, require := range plugin.Requires {
		_, requireFound := pluginsMap[require]
		if !requireFound {
			return errors.New("插件 " + slug + " 需要依赖 " + require + " 插件")
		}
	}

	for _, exclude := range plugin.Excludes {
		_, excludeFound := pluginsMap[exclude]
		if excludeFound {
			return errors.New("插件 " + slug + " 不兼容 " + exclude + " 插件")
		}
	}

	var task models.Task
	task.Name = "卸载插件 " + plugin.Name
	task.Status = models.TaskStatusWaiting
	task.Shell = plugin.Uninstall + " >> /tmp/" + plugin.Slug + ".log 2>&1"
	task.Log = "/tmp/" + plugin.Slug + ".log"
	if err = facades.Orm().Query().Create(&task); err != nil {
		return errors.New("创建任务失败")
	}

	r.task.Process(task.ID)
	return nil
}

// Update 更新插件
func (r *PluginImpl) Update(slug string) error {
	plugin := r.GetBySlug(slug)
	installedPlugin := r.GetInstalledBySlug(slug)
	installedPlugins, err := r.AllInstalled()
	if err != nil {
		return err
	}

	if installedPlugin.ID == 0 {
		return errors.New("插件未安装")
	}

	pluginsMap := make(map[string]bool)

	for _, p := range installedPlugins {
		pluginsMap[p.Slug] = true
	}

	for _, require := range plugin.Requires {
		_, requireFound := pluginsMap[require]
		if !requireFound {
			return errors.New("插件 " + slug + " 需要依赖 " + require + " 插件")
		}
	}

	for _, exclude := range plugin.Excludes {
		_, excludeFound := pluginsMap[exclude]
		if excludeFound {
			return errors.New("插件 " + slug + " 不兼容 " + exclude + " 插件")
		}
	}

	var task models.Task
	task.Name = "更新插件 " + plugin.Name
	task.Status = models.TaskStatusWaiting
	task.Shell = plugin.Update + " >> /tmp/" + plugin.Slug + ".log 2>&1"
	task.Log = "/tmp/" + plugin.Slug + ".log"
	if err = facades.Orm().Query().Create(&task); err != nil {
		return errors.New("创建任务失败")
	}

	r.task.Process(task.ID)
	return nil
}
