package vorm

import (
	"database/sql"
	"errors"
	"fmt"
	vexLog "github.com/axzed/vex/log"
	"reflect"
	"strings"
	"time"
)

type VexDb struct {
	db     *sql.DB        // db 维护一个数据库连接
	logger *vexLog.Logger // logger 维护一个日志对象
	Prefix string         // Prefix 维护一个表前缀
}

type VexSession struct {
	db          *VexDb          // db 维护一个数据库连接
	tableName   string          // tableName 维护一个表名
	fieldName   []string        // fieldName 维护字段名
	placeHolder []string        // placeHolder 维护占位符
	values      []any           // values 维护字段值
	updateParam strings.Builder // updateParam 维护更新参数
	whereParam  strings.Builder // whereParam 维护where条件参数
	whereValues []any           // whereValues 维护where条件值
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
		db:     db,
		logger: vexLog.Default(),
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
func (s *VexSession) Insert(data any) (int64, int64, error) {
	// insert into table (xxx, xxx) values (?, ?)
	s.fieldNames(data)
	// 拼接sql语句
	query := fmt.Sprintf("insert into %s (%s) values (%s)", s.tableName, strings.Join(s.fieldName, ","), strings.Join(s.placeHolder, ","))
	// 打印sql语句
	s.db.logger.Info(query)
	// prepare sql语句 用于后续的执行
	stmt, err := s.db.db.Prepare(query)
	if err != nil {
		return -1, -1, err
	}
	// 执行sql语句
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	// 影响的行数
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}

	return id, affected, nil
}

// InsertBatch 批量插入数据
func (s *VexSession) InsertBatch(data []any) (int64, int64, error) {
	// insert into table (xxx, xxx) values (?, ?), (?, ?), (?, ?)
	if len(data) == 0 {
		return -1, -1, errors.New("data is empty")
	}
	// 获取处理数据(结构体)的字段名
	s.fieldNames(data[0])
	// 拼接sql语句
	query := fmt.Sprintf("insert into %s (%s) values", s.tableName, strings.Join(s.fieldName, ","))
	var sb strings.Builder
	sb.WriteString(query)
	// 拼接占位符
	for index, _ := range data {
		sb.WriteString("(")
		sb.WriteString(strings.Join(s.placeHolder, ","))
		sb.WriteString(")")
		// 拼接逗号(不是最后一组数据)
		if index < len(data)-1 {
			sb.WriteString(",")
		}
	}
	// batchValues 获取处理数据(结构体)的字段值
	s.batchValues(data)
	query = sb.String()
	// 打印sql语句
	s.db.logger.Info(query)
	// prepare sql语句 用于后续的执行
	stmt, err := s.db.db.Prepare(query)
	if err != nil {
		return -1, -1, err
	}
	// 执行sql语句
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	// 影响的行数
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}

	return id, affected, nil
}

// Update 更新数据
func (s *VexSession) Update(data ...any) (int64, int64, error) {
	// update("age", 1) or update(user)
	// update table set xxx = ?, xxx = ? where xxx = ?
	if len(data) == 0 || len(data) > 2 {
		return -1, -1, errors.New("data is empty or data is too much")
	}
	single := true
	if len(data) == 2 {
		single = false
	}
	// update table set age = ?, name = ? where id = ?
	if !single {
		// 当 s.updateParam 为空时，不需要拼接逗号
		if s.updateParam.String() != "" {
			s.updateParam.WriteString(",")
		}
		// update("age", 1)
		s.updateParam.WriteString(data[0].(string))
		s.updateParam.WriteString(" = ? ")
		s.values = append(s.values, data[1])
	}
	query := fmt.Sprintf("update %s set %s", s.tableName, s.updateParam.String())
	// 拼接where条件
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	query = sb.String()
	// 打印sql语句
	s.db.logger.Info(query)
	// prepare sql语句 用于后续的执行
	stmt, err := s.db.db.Prepare(query)
	if err != nil {
		return -1, -1, err
	}
	// 拼接where条件的值
	s.values = append(s.values, s.whereValues...)
	// 执行sql语句
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	// 影响的行数
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}

	return id, affected, nil
}

// Where 条件查询语句字符串处理 where xxx = ?
func (s *VexSession) Where(field string, value any) *VexSession {
	// where xxx = ?
	if s.whereParam.String() == "" {
		// 第一次拼接where
		s.whereParam.WriteString("where ")
	} else {
		// 拼接and
		s.whereParam.WriteString("and ")
	}
	// 拼接字段名
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" = ")
	s.whereParam.WriteString("? ")
	// 拼接字段值
	s.whereValues = append(s.whereValues, value)
	return s
}

// filedNames 获取字段名
func (s *VexSession) fieldNames(data any) {
	// 获取传入的数据类型的反射类型和反射值
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	// 要求传入的数据类型必须是指针类型 例如: *User
	// 方便利用反射获取字段名和字段值
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data type must be pointer"))
	}
	// 获取指针指向的类型
	tVar := t.Elem()
	vVar := v.Elem()
	// 如果没有设置表名,则使用 prefix + 结构体 名作为表名
	if s.tableName == "" {
		s.tableName = s.db.Prefix + strings.ToLower(tVar.Name())
	}

	for i := 0; i < tVar.NumField(); i++ {
		// 获取字段名
		fieldName := tVar.Field(i).Name
		// 解析 Tag
		tag := tVar.Field(i).Tag
		sqlTag := tag.Get("vorm")
		// 没有设置vorm标签,则使用字段名默认匹配
		if sqlTag == "" {
			// UserName -> user_name
			sqlTag = strings.ToLower(Name(fieldName))
		} else {
			// 若设置了vorm的auto_increment,则不需要插入
			if strings.Contains(sqlTag, "auto_increment") {
				// 自增长的主键id
				continue
			}
			// 如果设置了vorm标签,则使用vorm标签匹配
			// 如果vorm标签中包含逗号,则只取逗号之前的内容
			if strings.Contains(sqlTag, ",") {
				sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
			}
		}
		id := vVar.Field(i).Interface()
		// 如果vorm标签中包含id,则判断是否是自增长的主键
		if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
			continue
		}
		s.fieldName = append(s.fieldName, sqlTag)
		s.placeHolder = append(s.placeHolder, "?")
		s.values = append(s.values, vVar.Field(i).Interface())
	}
}

// batchValues 获取处理数据(结构体)的字段值
func (s *VexSession) batchValues(data []any) {
	s.values = make([]any, 0)
	for _, value := range data {
		// 获取传入的数据类型的反射类型和反射值
		t := reflect.TypeOf(value)
		v := reflect.ValueOf(value)
		// 要求传入的数据类型必须是指针类型 例如: *User
		// 方便利用反射获取字段名和字段值
		if t.Kind() != reflect.Pointer {
			panic(errors.New("data type must be pointer"))
		}
		// 获取指针指向的类型
		tVar := t.Elem()
		vVar := v.Elem()

		for i := 0; i < tVar.NumField(); i++ {
			// 获取字段名
			fieldName := tVar.Field(i).Name
			// 解析 Tag
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("vorm")
			// 没有设置vorm标签,则使用字段名默认匹配
			if sqlTag == "" {
				// UserName -> user_name
				sqlTag = strings.ToLower(Name(fieldName))
			} else {
				// 若设置了vorm的auto_increment,则不需要插入
				if strings.Contains(sqlTag, "auto_increment") {
					// 自增长的主键id
					continue
				}
			}
			id := vVar.Field(i).Interface()
			// 如果vorm标签中包含id,则判断是否是自增长的主键
			if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
				continue
			}
			s.values = append(s.values, vVar.Field(i).Interface())
		}
	}
}

// IsAutoId 判断是否是自增长的id主键
func IsAutoId(id any) bool {
	t := reflect.TypeOf(id)
	v := reflect.ValueOf(id)
	switch t.Kind() {
	case reflect.Int64:
		if v.Interface().(int64) <= 0 {
			return true
		}
	case reflect.Int32:
		if v.Interface().(int32) <= 0 {
			return true
		}
	case reflect.Int:
		if v.Interface().(int) <= 0 {
			return true
		}
	default:
		return false
	}
	return false
}

// Name 将驼峰命名转换为下划线命名 例如: UserName -> User_Name
func Name(name string) string {
	var names = name[:]
	lastIndex := 0
	var sb strings.Builder
	for index, value := range names {
		// 判断是否是大写字母
		if value >= 65 && value <= 90 {
			// 如果是第一个字母,则不需要添加下划线
			if index == 0 {
				continue
			}
			sb.WriteString(name[:index])
			sb.WriteString("_")
			lastIndex = index
		}
	}
	if lastIndex != len(name)-1 {
		sb.WriteString(name[lastIndex:])
	}
	return sb.String()
}
