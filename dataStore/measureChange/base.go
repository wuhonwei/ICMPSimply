package measureChange

type DataModel struct {
	ID         uint   `gorm:"primary_key"`
	SourceIp   string `gorm:"column:source_ip;varchar(200)"`
	SourceName string `gorm:"column:source_name;primary_key;varchar(200);not null"`
	DesIp      string `gorm:"column:des_ip;varchar(80)"`
	DesName    string `gorm:"column:des_name;primary_key;varchar(200);not null"`
	// type 保留
	TimeStamp string `gorm:"column:timestamp;primary_key;varchar(80);not null"`
}
