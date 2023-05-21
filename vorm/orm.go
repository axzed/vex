package vorm

import (
	"database/sql"
	"time"
)

type VexDb struct {
	db *sql.DB
}

type VexSession struct {
	db        *VexDb
	tableName string
}

// Open 打开数据库连接,返回一个VexDb对象,用于操作数据库
// driverName: 驱动名称
// source: 数据库连接字符串
func Open(driverName string, source string) *VexDb {
	db, err := sql.Open(driverName, source)
	if err != nil {
		panic(err)
	}
	vexDb := &VexDb{
		db: db,
	}
	// 设置连接池 以下是vORM数据库连接池的默认配置
	// 最大空闲连接数，默认不配置，是2个最大空闲连接
	db.SetMaxIdleConns(5)
	// 最大连接数，默认不配置，是不限制最大连接数
	db.SetMaxOpenConns(100)
	// 连接最大存活时间
	db.SetConnMaxLifetime(time.Minute * 3)
	// 空闲连接最大存活时间
	db.SetConnMaxIdleTime(time.Minute * 1)
	// 检查连接是否有效
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	return vexDb
}

// SetMaxIdleConns 设置最大空闲连接数
func (d *VexDb) SetMaxIdleConns(n int) {
	d.db.SetMaxIdleConns(n)
}

// SetMaxOpenConns 设置最大连接数
func (d *VexDb) SetMaxOpenConns(n int) {
	d.db.SetMaxOpenConns(n)
}

// SetConnMaxLifetime 设置连接最大存活时间
func (d *VexDb) SetConnMaxLifetime(time time.Duration) {
	d.db.SetConnMaxLifetime(time)
}

// SetConnMaxIdleTime 设置空闲连接最大存活时间
func (d *VexDb) SetConnMaxIdleTime(time time.Duration) {
	d.db.SetConnMaxIdleTime(time)
}

// New 创建 VexSession 使得数据操作在一个会话内
func (d *VexDb) New() *VexSession {
	return &VexSession{
		db: d,
	}
}

// Table 指定本次 Session 要操作的数据库表名
func (s *VexSession) Table(name string) *VexSession {
	s.tableName = name
	return s
}

// Insert 插入数据
func (s *VexSession) Insert(data any) {
	// insert into table (xxx, xxx) values (?, ?)
}
