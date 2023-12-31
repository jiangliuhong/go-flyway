package history

import (
	"fmt"
	"github.com/jiangliuhong/go-flyway/consts"
	"github.com/jiangliuhong/go-flyway/database"
	"time"
)

type SchemaHistory struct {
	Database     database.Database
	Schema       database.Schema
	Table        database.Table
	BaseLineRank int
}

func (sh SchemaHistory) Exists() (bool, error) {
	return sh.Table.Exists()
}

func (sh SchemaHistory) Create() error {
	return sh.Table.Create()
}

func New(d database.Database, tableName string) (*SchemaHistory, error) {
	schema, err := d.CurrentSchema()
	if err != nil {
		return nil, err
	}
	if tableName == "" {
		tableName = consts.DEFAULT_HISTORY_TABLE
	}
	table, err := schema.Table(tableName)
	if err != nil {
		return nil, err
	}
	return &SchemaHistory{Database: d, Schema: schema, Table: table}, nil
}

func (sh SchemaHistory) getBaseQuery() string {
	return fmt.Sprintf(`select * from %s.%s `, sh.Schema.Name(), sh.Table.Name())
}

func (sh *SchemaHistory) InitBaseLineRank() error {

	querySql := sh.getBaseQuery() + ` where type = ? `
	var sd SchemaData
	exist, err := sh.Database.Session().SelectOne(querySql, &sd, consts.BASE_LINE_TYPE)
	if err != nil {
		return err
	}
	if !exist {
		// 不存在新增一条
		user, err := sh.Database.CurrentUser()
		if err != nil {
			return err
		}
		sd = SchemaData{
			Version:       "1",
			Description:   consts.BASE_LINE_DESC,
			Type:          consts.BASE_LINE_TYPE,
			Script:        consts.BASE_LINE_DESC,
			InstalledBy:   user,
			InstalledOn:   time.Now().Format("2006-01-02 15:04:05"),
			ExecutionTime: 0,
			Success:       true,
		}
		rank, err := sh.InsertData(sd)
		if err != nil {
			return err
		}
		sh.BaseLineRank = rank
	}
	return nil
}

func (sh SchemaHistory) InsertData(sd SchemaData) (newRank int, err error) {
	sql := sh.getBaseQuery() + ` where installed_rank > ? `
	var querySd SchemaData
	newRank = 1
	exist, err := sh.Database.Session().SelectOne(sql, &querySd, sh.BaseLineRank)
	if err != nil {
		return
	}

	if exist {
		newRank = querySd.InstalledRank + 1
	}
	insertSql := fmt.Sprintf(`insert into %s.%s values (?,?,?,?,?,?,?,?,?,?)`,
		sh.Schema.Name(), sh.Table.Name())
	_, err = sh.Database.Session().Insert(insertSql,
		newRank,
		sd.Version,
		sd.Description,
		sd.Type,
		sd.Script,
		sd.Checksum,
		sd.InstalledBy,
		sd.InstalledOn,
		sd.ExecutionTime,
		sd.Success)
	return
}

func (sh SchemaHistory) SelectVersion(version string) (*SchemaData, error) {
	sql := sh.getBaseQuery() + ` where version = ? and installed_rank > ? order by execution_time desc limit 1`
	var sd SchemaData
	exist, err := sh.Database.Session().SelectOne(sql, &sd, version, sh.BaseLineRank)
	if err != nil {
		return nil, err
	}
	if exist {
		return &sd, nil
	}
	return nil, nil
}

func (sh SchemaHistory) GetLatestRank() (int, error) {
	sql := sh.getBaseQuery() + ` order by installed_rank desc limit 1`
	var sd SchemaData
	exists, err := sh.Database.Session().SelectOne(sql, &sd)
	if err != nil {
		return 0, err
	}
	if exists {
		return sd.InstalledRank, nil
	}
	return 0, nil
}
