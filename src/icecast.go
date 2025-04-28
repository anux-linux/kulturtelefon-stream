package main

import (
	"fmt"
	"os"
	"path/filepath"
	"text/template"
)

type IcecastConfigStore struct {
	config Config
}

type TemplateType string

const (
	DefaultTemplate TemplateType = "default"
	PrivateTemplate TemplateType = "private"
)

func NewIcecastConfig(config Config) *IcecastConfigStore {
	return &IcecastConfigStore{
		config: config,
	}
}

func (icConf *IcecastConfigStore) getMountConfigTemplate(templateType TemplateType) string {
	// Get the mount configuration template based on the template type
	switch templateType {
	case DefaultTemplate:
		return icConf.config.DefaultMountTemplate
	case PrivateTemplate:
		return icConf.config.PrivateMountTemplate
	default:
		return icConf.config.DefaultMountTemplate
	}
}

func (icConf *IcecastConfigStore) SaveMountConfig(mount IcecastMount) error {

	// Get the mount configuration template
	templateFile := icConf.getMountConfigTemplate(mount.TemplateType)
	mountsDirectory := icConf.config.IcecastMountsFolder
	fielname := icConf.getMountConfigFileName(mount.MountName, mount.TemplateType)

	if !checkFileExists(templateFile) {
		logWithCaller(fmt.Sprintf("Template file does not exist: %s", templateFile), FatalLog)
		return fmt.Errorf("femplate file does not exist: %s", templateFile)
	}
	if !checkDirectoryExists(mountsDirectory) {
		logWithCaller(fmt.Sprintf("Mounts directory does not exist: %s", mountsDirectory), FatalLog)
		return fmt.Errorf("mounts directory file does not exist: %s", mountsDirectory)
	}

	tmpl, err := template.ParseFiles(templateFile)

	if err != nil {
		logWithCaller(fmt.Sprintf("Error parsing template file: %s", err), FatalLog)
		return err
	}

	filePath := mountsDirectory + "/" + fielname

	file, err := os.Create(filePath)
	if err != nil {

		logWithCaller(fmt.Sprintf("Error creating mount configuration file: %s", err), FatalLog)
		return fmt.Errorf("error creating mount configuration file: %s", err)
	}
	defer file.Close()

	// Get the template name (filename without extension)
	templateName := filepath.Base(templateFile)
	err = tmpl.ExecuteTemplate(file, templateName, mount)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error executing template: %s", err), FatalLog)
		return fmt.Errorf("error executing template: %s", err)
	}
	// Set the file permissions to 0644
	err = os.Chmod(filePath, 0644)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error setting file permissions: %s", err), FatalLog)
		return fmt.Errorf("error setting file permissions: %s", err)
	}
	return nil
}

func (icConf *IcecastConfigStore) DeleteMountConfig(mount IcecastMount) error {

	mountsDirectory := icConf.config.IcecastMountsFolder
	// Delete the mount configuration file
	filePath := mountsDirectory + "/" + icConf.getMountConfigFileName(mount.MountName, mount.TemplateType)
	err := os.Remove(filePath)
	if err != nil {
		logWithCaller(fmt.Sprintf("Error deleting mount configuration file:  %s", err), FatalLog)
		return fmt.Errorf("error deleting mount configuration file: %s", err)
	}
	return nil
}

func (icConf *IcecastConfigStore) getMountConfigFileName(mountName string, templateType TemplateType) string {
	// Get the mount configuration file name
	return mountName + "-" + string(templateType) + ".xml"
}
