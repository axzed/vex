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
func (d *VexDb) New(data any) *VexSession {
	m := &VexSession{
		db: d,
	}
	// 获取传入的数据类型的反射类型和反射值
	t := reflect.TypeOf(data)
	// 要求传入的数据类型必须是指针类型 例如: *User
	// 方便利用反射获取字段名和字段值
	if t.Kind() != reflect.Pointer {
		panic(errors.New("data type must be pointer"))
	}
	// 获取指针指向的类型
	tVar := t.Elem()
	// 如果没有设置表名,则使用 prefix + 结构体 名作为表名
	if m.tableName == "" {
		m.tableName = m.db.Prefix + strings.ToLower(tVar.Name())
	}
	return m
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

// UpdateParam 链式调用更新多个数据的字段的值
func (s *VexSession) UpdateParam(field string, value any) *VexSession {
	// Update("age", 1).Update("name", "vex")
	// update table set xxx = ?, xxx = ? where xxx = ?
	// 当 s.updateParam 不为空时，需要拼接逗号
	if s.updateParam.String() != "" {
		s.updateParam.WriteString(",")
	}
	// 拼接sql语句
	// 拼接字段名
	s.updateParam.WriteString(field)
	// 拼接占位符
	s.updateParam.WriteString(" = ?")
	// 拼接字段值
	s.values = append(s.values, value)
	return s
}

// UpdateMap 链式调用更新多个数据的字段的值(用map标识要更新的数据)
// map[xxx] = ?
func (s *VexSession) UpdateMap(data map[string]any) *VexSession {
	// Update("age", 1).Update("name", "vex")
	// update table set xxx = ?, xxx = ? where xxx = ?

	// 遍历map 获取字段名和字段值
	for k, v := range data {
		// 当 s.updateParam 不为空时，需要拼接逗号
		if s.updateParam.String() != "" {
			s.updateParam.WriteString(",")
		}
		// 拼接sql语句
		// 拼接字段名
		s.updateParam.WriteString(k)
		// 拼接占位符
		s.updateParam.WriteString(" = ?")
		// 拼接字段值
		s.values = append(s.values, v)
	}

	return s
}

// Update 更新数据
func (s *VexSession) Update(data ...any) (int64, int64, error) {
	// update("age", 1) or update(user)
	// update table set xxx = ?, xxx = ? where xxx = ?
	if len(data) > 2 {
		return -1, -1, errors.New("data is empty or data is too much")
	}
	// 当 data 为空时候，代表前面已经结果链式 update 对 sql 进行了处理
	// 直接执行sql语句
	if len(data) == 0 {
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
	} else {
		// data[0] 是结构体用来更新
		updateData := data[0]
		// 获取传入的数据类型的反射类型和反射值
		t := reflect.TypeOf(updateData)
		v := reflect.ValueOf(updateData)
		// 要求传入的数据类型必须是指针类型 例如: *User
		// 方便利用反射获取字段名和字段值
		if t.Kind() != reflect.Pointer {
			panic(errors.New("update data type must be pointer"))
		}
		// 获取指针指向的类型
		tVar := t.Elem()
		vVar := v.Elem()
		// 如果没有设置表名,则使用 prefix + 结构体 名作为表名
		if s.tableName == "" {
			s.tableName = s.db.Prefix + strings.ToLower(tVar.Name())
		}

		// 遍历结构体的字段 获取字段名和字段值 拼接sql语句 例如: update table set age = ?, name = ? where id = ?
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
			// 当 s.updateParam 不为空时，需要拼接逗号
			if s.updateParam.String() != "" {
				s.updateParam.WriteString(",")
			}
			// update(user) 传入的是结构体 例如: user := User{Id: 1, Age: 18, Name: "vex"}
			// update table set age = ?, name = ? where id = ?
			// 将字段名添加到对应的切片中
			s.updateParam.WriteString(sqlTag)
			// 拼接展位符
			s.updateParam.WriteString(" = ? ")
			// 将字段值添加到切片中
			s.values = append(s.values, vVar.Field(i).Interface())
		}
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

// SelectOne 查询一条数据
func (s *VexSession) SelectOne(data any, fields ...string) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr {
		return errors.New("data must be a pointer")
	}
	// 默认查询字段是 *
	fieldStr := "*"
	if len(fields) > 0 {
		// 传入了查询字段
		fieldStr = strings.Join(fields, ",")
	}
	query := fmt.Sprintf("select %s from %s ", fieldStr, s.tableName)
	// 拼接where条件
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	query = sb.String()
	s.db.logger.Info(query)

	// prepare sql语句 用于后续的执行
	stmt, err := s.db.db.Prepare(query)
	if err != nil {
		return err
	}
	rows, err := stmt.Query(s.whereValues...)
	if err != nil {
		return err
	}

	// 获取查询的字段名
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	// id username age
	values := make([]any, len(columns))    // values是每个列的值，这里获取到byte里
	fieldScan := make([]any, len(columns)) // fieldScan是每个scan的值，这里获取到[]interface{}里
	// 将字段名和字段值对应起来
	for i := range fieldScan {
		fieldScan[i] = &values[i]
	}

	// 判断是否有数据
	if rows.Next() {
		// 扫描数据 将数据赋值给fieldScan
		err := rows.Scan(fieldScan...)
		if err != nil {
			return err
		}
		// 将数据赋值给data
		tVar := t.Elem()
		vVar := reflect.ValueOf(data).Elem()
		// 遍历字段名 获取字段名对应的值 并赋值给data
		for i := 0; i < tVar.NumField(); i++ {
			name := tVar.Field(i).Name              // 获取字段名
			sqlTag := tVar.Field(i).Tag.Get("vorm") // 获取tag
			// 判断tag是否为空
			if sqlTag == "" {
				// tag为空，将字段名转为小写 作为sql语句中的字段名 例如：id UserName Age 转为 id user_name age
				sqlTag = strings.ToLower(Name(name))
			} else {
				// 如果tag不为空，判断是否有逗号
				if strings.Contains(sqlTag, ",") {
					// 有逗号，截取第一个逗号前的内容作为sql语句中的字段名
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}

			// 遍历从数据库中查询出来的字段名 获取字段名对应的值 并赋值给data
			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]                    // 获取字段名对应的值
					targetValue := reflect.ValueOf(target) // 获取字段名对应的值的反射值
					fieldType := tVar.Field(i).Type
					// 类型转换
					result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType) // 将字段名对应的值的反射值转换为字段类型
					vVar.Field(i).Set(result)                                             // 将字段名对应的值的反射值转换后的值赋值给data
				}
			}
		}
	}
	return nil
}

// Select 查询多行数据
func (s *VexSession) Select(data any, fields ...string) ([]any, error) {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr {
		return nil, errors.New("data must be a pointer")
	}
	// 默认查询字段是 *
	fieldStr := "*"
	if len(fields) > 0 {
		// 传入了查询字段
		fieldStr = strings.Join(fields, ",")
	}
	query := fmt.Sprintf("select %s from %s ", fieldStr, s.tableName)
	// 拼接where条件
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	query = sb.String()
	s.db.logger.Info(query)

	// prepare sql语句 用于后续的执行
	stmt, err := s.db.db.Prepare(query)
	if err != nil {
		return nil, err
	}
	rows, err := stmt.Query(s.whereValues...)
	if err != nil {
		return nil, err
	}

	// 获取查询的字段名
	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	result := make([]any, 0)

	// 判断是否有数据 (与单行查询不同的是，这里需要循环遍历所有的数据)
	for {
		if rows.Next() {
			// 由于传入进来的是一个指针地址 如果每次赋值都是同一个地址
			// 所以每次查询的时候都需要重新创建一个新的地址
			data := reflect.New(t.Elem()).Interface()
			// id username age
			values := make([]any, len(columns))    // values是每个列的值，这里获取到byte里
			fieldScan := make([]any, len(columns)) // fieldScan是每个scan的值，这里获取到[]interface{}里
			// 将字段名和字段值对应起来
			for i := range fieldScan {
				fieldScan[i] = &values[i]
			}
			// 扫描数据 将数据赋值给fieldScan
			err := rows.Scan(fieldScan...)
			if err != nil {
				return nil, err
			}
			// 将数据赋值给data
			tVar := t.Elem()
			vVar := reflect.ValueOf(data).Elem()
			// 遍历字段名 获取字段名对应的值 并赋值给data
			for i := 0; i < tVar.NumField(); i++ {
				name := tVar.Field(i).Name              // 获取字段名
				sqlTag := tVar.Field(i).Tag.Get("vorm") // 获取tag
				// 判断tag是否为空
				if sqlTag == "" {
					// tag为空，将字段名转为小写 作为sql语句中的字段名 例如：id UserName Age 转为 id user_name age
					sqlTag = strings.ToLower(Name(name))
				} else {
					// 如果tag不为空，判断是否有逗号
					if strings.Contains(sqlTag, ",") {
						// 有逗号，截取第一个逗号前的内容作为sql语句中的字段名
						sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
					}
				}

				// 遍历从数据库中查询出来的字段名 获取字段名对应的值 并赋值给data
				for j, colName := range columns {
					if sqlTag == colName {
						target := values[j]                    // 获取字段名对应的值
						targetValue := reflect.ValueOf(target) // 获取字段名对应的值的反射值
						fieldType := tVar.Field(i).Type
						// 类型转换
						result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType) // 将字段名对应的值的反射值转换为字段类型
						vVar.Field(i).Set(result)                                             // 将字段名对应的值的反射值转换后的值赋值给data
					}
				}
			}
			// 将每次赋值好的data追加到result中
			result = append(result, data)
		} else {
			break
		}
	}

	return result, nil
}

// Like 模糊查询
func (s *VexSession) Like(field string, data any) *VexSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" like ?")

	s.values = append(s.values, "%"+data.(string)+"%")
	return s
}

// LikeRight 模糊查询 右匹配
func (s *VexSession) LikeRight(field string, data any) *VexSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" where ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" like ?")

	s.values = append(s.values, data.(string)+"%")
	return s
}

func (s *VexSession) Group(field ...string) *VexSession {
	s.whereParam.WriteString(" group by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	return s
}

func (s *VexSession) OrderDesc(field ...string) *VexSession {
	s.whereParam.WriteString(" order by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	s.whereParam.WriteString(" desc ")
	return s
}

func (s *VexSession) OrderAsc(field ...string) *VexSession {
	s.whereParam.WriteString(" order by ")
	s.whereParam.WriteString(strings.Join(field, ","))
	s.whereParam.WriteString(" asc ")
	return s
}

// Order // order by name asc,age desc
func (s *VexSession) Order(field ...string) *VexSession {
	s.whereParam.WriteString(" order by ")
	size := len(field)
	if size%2 != 0 {
		panic("Order field must be 偶数")
	}
	for index, v := range field {
		s.whereParam.WriteString(" ")
		s.whereParam.WriteString(v)
		s.whereParam.WriteString(" ")
		if index%2 != 0 && index < len(field)-1 {
			s.whereParam.WriteString(",")
		}
	}
	return s
}

// Count 聚合函数
func (s *VexSession) Count(field string) (int64, error) {
	return s.Aggregate("count", field)
}

// Sum 聚合函数
func (s *VexSession) Sum(field string) (int64, error) {
	return s.Aggregate("sum", field)
}

// Max 聚合函数
func (s *VexSession) Max(field string) (int64, error) {
	return s.Aggregate("max", field)
}

// Min 聚合函数
func (s *VexSession) Min(field string) (int64, error) {
	return s.Aggregate("min", field)
}

// Avg 聚合函数
func (s *VexSession) Avg(field string) (int64, error) {
	return s.Aggregate("avg", field)
}

// Aggregate 聚合函数 sum max min avg
func (s *VexSession) Aggregate(funcName, field string) (int64, error) {
	// select sum(field) from tableName where xxx = ?
	// 拼接聚合函数 select sum(field) from tableName
	var aggSb strings.Builder
	aggSb.WriteString(funcName)
	aggSb.WriteString("(")
	aggSb.WriteString(field)
	aggSb.WriteString(")")
	// 拼接select语句
	query := fmt.Sprintf("select %s from %s ", aggSb.String(), s.tableName)
	// 拼接where语句
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	s.db.logger.Info(sb.String())
	// 执行sql
	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return 0, err
	}
	var result int64
	row := stmt.QueryRow()
	err = row.Err()
	if err != nil {
		return 0, err
	}
	err = row.Scan(&result)
	if err != nil {
		return 0, err
	}
	return result, nil
}

// Delete 删除数据
func (s *VexSession) Delete() (int64, error) {
	// delete from tableName where xxx = ?
	// 拼接delete语句
	query := fmt.Sprintf("delete from %s ", s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	query = sb.String()
	s.db.logger.Info(query)

	// prepare sql语句 用于后续的执行
	stmt, err := s.db.db.Prepare(query)
	if err != nil {
		return 0, err
	}
	r, err := stmt.Exec(s.whereValues...)
	if err != nil {
		return 0, err
	}
	return r.RowsAffected()
}

// Where 条件查询语句字符串处理 where xxx = ?
func (s *VexSession) Where(field string, value any) *VexSession {
	// where xxx = ?
	if s.whereParam.String() == "" {
		// 第一次拼接where
		s.whereParam.WriteString("where ")
	}
	// 拼接字段名
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" = ")
	s.whereParam.WriteString("? ")
	// 拼接字段值
	s.whereValues = append(s.whereValues, value)
	return s
}

// And 多条件查询语句字符串处理 and xxx = ?
func (s *VexSession) And() *VexSession {
	// 拼接and
	s.whereParam.WriteString("and ")
	return s
}

// Or 条件查询语句字符串处理 where xxx = ?
func (s *VexSession) Or() *VexSession {
	s.whereParam.WriteString("or ")
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
