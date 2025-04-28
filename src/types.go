package main

type Config struct {
	IcecastMountsFolder  string `yaml:"icecast_mounts_folder"`
	DbFile               string `yaml:"db_file"`
	DefaultMountTemplate string `yaml:"default_mount_template"`
	PrivateMountTemplate string `yaml:"private_mount_template"`
	SecretKey            string `yaml:secret_key"`
	AdminUsername        string `yaml:"admin_username"`
	AdminPassword        string `yaml:"admin_password"`
}

// IcecastMount represents the configuration for an Icecast mount point
type IcecastMount struct {
	MountName         string       `json:"mount_name"`
	Username          string       `json:"username"`
	Password          string       `json:"password"`
	Public            int          `json:"public"`
	StreamName        string       `json:"stream_name"`
	StreamDescription string       `json:"stream_description"`
	TemplateType      TemplateType `json:"template_type"`
}
