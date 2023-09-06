package orm

import (
	"database/sql"
	"errors"
	"fmt"
	nxjLog "github.com/Komorebi695/nxjgo/log"
	"reflect"
	"strings"
	"time"
)

type NxjDb struct {
	db     *sql.DB
	logger *nxjLog.Logger
	Prefix string
}

type NxjSession struct {
	db          *NxjDb
	tx          *sql.Tx
	beginTx     bool
	tableName   string
	fieldName   []string
	placeHolder []string
	values      []any
	updateParam strings.Builder
	whereParam  strings.Builder
	whereValue  []any
	limitValue  int64
	offsetValue int64
}

func Open(driverName string, source string) (*NxjDb, error) {
	db, err := sql.Open(driverName, source)
	if err != nil {
		panic(err)
	}
	nxjDb := &NxjDb{
		db:     db,
		logger: nxjLog.Default(),
	}
	// 最大空闲连接数，默认不配置，是2个最大空闲连接
	db.SetMaxIdleConns(5)
	// 最大连接数，默认不配置，是不限制最大连接数
	//db.SetMaxOpenConns(100)
	// 连接最大存活时间
	db.SetConnMaxLifetime(time.Minute * 3)
	// 空闲连接最大存活时间
	db.SetConnMaxIdleTime(time.Minute * 1)
	err = db.Ping()
	if err != nil {
		panic(err)
	}
	return nxjDb, nil
}

func (db *NxjDb) Close() error {
	return db.db.Close()
}

// SetMaxIdleConns 设置最大空闲连接数
func (db *NxjDb) SetMaxIdleConns(n int) {
	db.db.SetMaxIdleConns(n)
}

// SetSetMaxOpenConns 最大连接数，默认不配置，是不限制最大连接数
func (db *NxjDb) SetSetMaxOpenConns(n int) {
	db.db.SetMaxOpenConns(n)
}

// SetConnMaxLifetime 连接最大存活时间
func (db *NxjDb) SetConnMaxLifetime(d time.Duration) {
	db.db.SetConnMaxLifetime(d)
}

// SetConnMaxIdleTime 空闲连接最大存活时间
func (db *NxjDb) SetConnMaxIdleTime(d time.Duration) {
	db.db.SetConnMaxIdleTime(d)
}

func (db *NxjDb) New() *NxjSession {
	return &NxjSession{
		db: db,
	}
}

// Table 设置表名称
func (s *NxjSession) Table(name string) *NxjSession {
	s.tableName = name
	return s
}

func (s *NxjSession) Insert(data any) (int64, int64, error) {
	// 每个操作都是独立的，互不影响的 session
	// insert into table (x,x,x,x) values(?..?);
	s.fieldNames(data)
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s);", s.tableName, strings.Join(s.fieldName, ","), strings.Join(s.placeHolder, ","))
	s.db.logger.Info(query)
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(query)
	} else {
		stmt, err = s.db.db.Prepare(query)
	}
	if err != nil {
		return -1, -1, err
	}
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}

func (s *NxjSession) fieldNames(data any) {
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if t.Kind() != reflect.Ptr {
		panic(errors.New("data must be pointer"))
	}
	tVar := t.Elem()
	vVar := v.Elem()
	if s.tableName == "" {
		s.tableName = s.db.Prefix + strings.ToLower(NameFormat(tVar.Name()))
	}
	for i := 0; i < tVar.NumField(); i++ {
		fieldName := tVar.Field(i).Name
		tag := tVar.Field(i).Tag
		sqlTag := tag.Get("norm")
		if sqlTag == "" {
			sqlTag = strings.ToLower(NameFormat(fieldName))
		} else {
			if strings.Contains(sqlTag, "auto_increment") {
				continue
			}
			if strings.Contains(sqlTag, ",") {
				sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
			}
		}
		id := vVar.Field(i).Interface()
		if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
			continue
		}
		s.fieldName = append(s.fieldName, sqlTag)
		s.placeHolder = append(s.placeHolder, "?")
		s.values = append(s.values, vVar.Field(i).Interface())
	}
}

func (s *NxjSession) InsertBatch(data []any) (int64, int64, error) {
	// insert into table (xxx,xxx) values(?,?),(?,?);
	if len(data) == 0 {
		return -1, -1, errors.New("no data insert")
	}
	s.fieldNames(data[0])
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES ", s.tableName, strings.Join(s.fieldName, ","))
	var sb strings.Builder
	sb.WriteString(query)
	for i, v := range data {
		sb.WriteString("(")
		sb.WriteString(strings.Join(s.placeHolder, ","))
		sb.WriteString(")")
		if i < len(data)-1 {
			sb.WriteString(",")
		}
		if i > 0 {
			s.batchValue(v)
		}
	}
	sb.WriteString(";")
	query = sb.String()
	s.db.logger.Info(query)
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(query)
	} else {
		stmt, err = s.db.db.Prepare(query)
	}
	if err != nil {
		return -1, -1, err
	}
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}

func (s *NxjSession) batchValue(data any) {
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if t.Kind() != reflect.Ptr {
		panic(errors.New("data must be pointer"))
	}
	tVar := t.Elem()
	vVar := v.Elem()
	if s.tableName == "" {
		s.tableName = s.db.Prefix + strings.ToLower(NameFormat(tVar.Name()))
	}
	for i := 0; i < tVar.NumField(); i++ {
		fieldName := tVar.Field(i).Name
		sqlTag := tVar.Field(i).Tag.Get("norm")
		if sqlTag == "" {
			sqlTag = strings.ToLower(NameFormat(fieldName))
		} else {
			if strings.Contains(sqlTag, "auto_increment") {
				continue
			}
		}
		id := vVar.Field(i).Interface()
		if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
			continue
		}
		s.values = append(s.values, vVar.Field(i).Interface())
	}
}

func (s *NxjSession) batchValues2(data []any) {
	s.values = make([]any, 0)
	for _, value := range data {
		t := reflect.TypeOf(value)
		v := reflect.ValueOf(value)
		if t.Kind() != reflect.Ptr {
			panic(errors.New("data must be pointer"))
		}
		tVar := t.Elem()
		vVar := v.Elem()
		if s.tableName == "" {
			s.tableName = s.db.Prefix + strings.ToLower(NameFormat(tVar.Name()))
		}
		for i := 0; i < tVar.NumField(); i++ {
			fieldName := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("norm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(NameFormat(fieldName))
			} else {
				if strings.Contains(sqlTag, "auto_increment") {
					continue
				}
			}
			id := vVar.Field(i).Interface()
			if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
				continue
			}
			s.values = append(s.values, vVar.Field(i).Interface())
		}
	}
}

func IsAutoId(id any) bool {
	t := reflect.TypeOf(id)
	switch t.Kind() {
	case reflect.Int64:
		if id.(int64) <= 0 {
			return true
		}
	case reflect.Int32:
		if id.(int32) <= 0 {
			return true
		}
	case reflect.Int:
		if id.(int) <= 0 {
			return true
		}
	default:
		return false
	}
	return false
}

func NameFormat(name string) string {
	var names = name[:]
	lastIndex := 0
	var sb strings.Builder
	for i, v := range names {
		if v >= 65 && v <= 90 {
			// 大写字母
			if i == 0 {
				continue
			}
			sb.WriteString(name[lastIndex:i])
			sb.WriteString("_")
			lastIndex = i
		}
	}
	sb.WriteString(name[lastIndex:])
	return sb.String()
}

func (s *NxjSession) Update(column string, value any) (int64, int64, error) {
	// update table set xxx = ?,xxx = ? where id = ?;
	if s.updateParam.String() != "" {
		s.updateParam.WriteString(",")
	}
	s.updateParam.WriteString(column)
	s.updateParam.WriteString(" = ? ")
	s.values = append(s.values, value)
	query := fmt.Sprintf("UPDATE %s SET %s", s.tableName, s.updateParam.String())
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	sb.WriteString(";")
	query = sb.String()
	s.db.logger.Info(query)
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(query)
	} else {
		stmt, err = s.db.db.Prepare(query)
	}
	if err != nil {
		return -1, -1, err
	}
	s.values = append(s.values, s.whereValue...)
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}

func (s *NxjSession) Updates(data any) (int64, int64, error) {
	// update table set xxx = ?,xxx = ? where id = ?;
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if t.Kind() != reflect.Ptr {
		panic(errors.New("data must be pointer"))
	}
	tVar := t.Elem()
	vVar := v.Elem()
	if s.tableName == "" {
		s.tableName = s.db.Prefix + strings.ToLower(NameFormat(tVar.Name()))
	}
	for i := 0; i < tVar.NumField(); i++ {
		fieldName := tVar.Field(i).Name
		tag := tVar.Field(i).Tag
		sqlTag := tag.Get("norm")
		if sqlTag == "" {
			sqlTag = strings.ToLower(NameFormat(fieldName))
		} else {
			if strings.Contains(sqlTag, "auto_increment") {
				continue
			}
			if strings.Contains(sqlTag, ",") {
				sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
			}
		}
		id := vVar.Field(i).Interface()
		if strings.ToLower(sqlTag) == "id" && IsAutoId(id) {
			continue
		}
		if s.updateParam.String() != "" {
			s.updateParam.WriteString(",")
		}
		s.updateParam.WriteString(sqlTag)
		s.updateParam.WriteString(" = ?")
		s.values = append(s.values, vVar.Field(i).Interface())
	}
	query := fmt.Sprintf("UPDATE %s SET %s", s.tableName, s.updateParam.String())
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	sb.WriteString(";")
	query = sb.String()
	s.db.logger.Info(query)
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(query)
	} else {
		stmt, err = s.db.db.Prepare(query)
	}
	if err != nil {
		return -1, -1, err
	}
	s.values = append(s.values, s.whereValue...)
	r, err := stmt.Exec(s.values...)
	if err != nil {
		return -1, -1, err
	}
	id, err := r.LastInsertId()
	if err != nil {
		return -1, -1, err
	}
	affected, err := r.RowsAffected()
	if err != nil {
		return -1, -1, err
	}
	return id, affected, nil
}

func (s *NxjSession) Where(field, condition string, value any) *NxjSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" WHERE ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(fmt.Sprintf(" %s ", condition))
	s.whereParam.WriteString("?")
	s.whereValue = append(s.whereValue, value)
	return s
}

func (s *NxjSession) In(field string, values ...any) *NxjSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" WHERE ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" IN (")
	for i, v := range values {
		s.whereParam.WriteString("?")
		if i < len(values)-1 {
			s.whereParam.WriteString(",")
		} else {
			s.whereParam.WriteString(")")
		}
		s.whereValue = append(s.whereValue, v)
	}
	return s
}

func (s *NxjSession) Between(field string, v1 any, v2 any) *NxjSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" WHERE ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" BETWEEN ")
	s.whereParam.WriteString("?")
	s.whereParam.WriteString(" AND ")
	s.whereParam.WriteString("?")
	s.whereValue = append(s.whereValue, v1, v2)
	return s
}

func (s *NxjSession) IsNull(field string) *NxjSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" WHERE ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" IS NULL")
	return s
}

func (s *NxjSession) Like(field string, value any) *NxjSession {
	if s.whereParam.String() == "" {
		s.whereParam.WriteString(" WHERE ")
	}
	s.whereParam.WriteString(field)
	s.whereParam.WriteString(" LIKE ")
	s.whereParam.WriteString("?")
	s.whereValue = append(s.whereValue, value)
	return s
}

func (s *NxjSession) And() *NxjSession {
	s.whereParam.WriteString(" AND ")
	return s
}

func (s *NxjSession) Or() *NxjSession {
	s.whereParam.WriteString(" OR ")
	return s
}

func (s *NxjSession) SelectOne(data any, fields ...string) error {
	t := reflect.TypeOf(data)
	v := reflect.ValueOf(data)
	if t.Kind() != reflect.Ptr {
		panic(errors.New("data must be pointer"))
	}
	tVar := t.Elem()
	vVar := v.Elem()
	if s.tableName == "" {
		s.tableName = s.db.Prefix + strings.ToLower(NameFormat(tVar.Name()))
	}
	fieldStr := "*"
	if len(fields) > 0 {
		fieldStr = strings.Join(fields, ",")
	}
	query := fmt.Sprintf("SELECT %s FROM %s", fieldStr, s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	if s.limitValue > 0 {
		sb.WriteString(" LIMIT ? OFFSET ?")
		s.whereValue = append(s.whereValue, s.limitValue, s.offsetValue)
	}
	sb.WriteString(";")
	s.db.logger.Info(sb.String())
	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return err
	}
	rows, err := stmt.Query(s.whereValue...)
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	values := make([]any, len(columns))
	fieldScan := make([]any, len(columns))
	for i := range fieldScan {
		fieldScan[i] = &values[i]
	}
	if rows.Next() {
		err = rows.Scan(fieldScan...)
		if err != nil {
			return err
		}
		for i := 0; i < tVar.NumField(); i++ {
			name := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("norm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(NameFormat(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]
					targetValue := reflect.ValueOf(target)
					fieldType := tVar.Field(i).Type
					result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
					vVar.Field(i).Set(result)
				}
			}
		}
	}
	return nil
}

func (s *NxjSession) Select(data any, fields ...string) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Slice {
		return errors.New("data must be a pointer to a slice")
	}
	tVar := t.Elem()
	if s.tableName == "" {
		s.tableName = s.db.Prefix + strings.ToLower(NameFormat(tVar.Name()))
	}
	fieldStr := "*"
	if len(fields) > 0 {
		fieldStr = strings.Join(fields, ",")
	}
	query := fmt.Sprintf("SELECT %s FROM %s", fieldStr, s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	if s.limitValue > 0 {
		sb.WriteString(" LIMIT ? OFFSET ?")
		s.whereValue = append(s.whereValue, s.limitValue, s.offsetValue)
	}
	sb.WriteString(";")
	s.db.logger.Info(sb.String())
	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return err
	}
	rows, err := stmt.Query(s.whereValue...)
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	t = t.Elem().Elem()
	v := reflect.ValueOf(data).Elem()
	for rows.Next() {
		newElem := reflect.New(t)
		values := make([]any, len(columns))
		fieldScan := make([]any, len(columns))
		for i := range fieldScan {
			fieldScan[i] = &values[i]
		}
		err = rows.Scan(fieldScan...)
		if err != nil {
			return err
		}
		for i := 0; i < t.NumField(); i++ {
			name := t.Field(i).Name
			tag := t.Field(i).Tag
			sqlTag := tag.Get("norm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(NameFormat(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]
					targetValue := reflect.ValueOf(target)
					fieldType := t.Field(i).Type
					res := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
					newElem.Elem().FieldByName(name).Set(res)
				}
			}
		}
		newSlice := reflect.Append(v, newElem.Elem())
		v.Set(newSlice)
	}
	return nil
}

func (s *NxjSession) Delete() (int64, error) {
	// delete from table where id = ?;
	query := fmt.Sprintf("DELETE FROM %s ", s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	sb.WriteString(";")
	s.db.logger.Info(sb.String())
	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return -1, err
	}
	r, err := stmt.Exec(s.whereValue...)
	if err != nil {
		return -1, err
	}
	return r.RowsAffected()
}

func (s *NxjSession) Group(field ...string) *NxjSession {
	// group by a,b
	s.whereParam.WriteString(" GROUP BY ")
	s.whereParam.WriteString(strings.Join(field, ","))
	return s
}

func (s *NxjSession) Order(field ...string) *NxjSession {
	// order by a,b
	s.whereParam.WriteString(" ORDER BY ")
	s.whereParam.WriteString(strings.Join(field, ","))
	return s
}

func (s *NxjSession) OrderDesc(field ...string) *NxjSession {
	// order by a,b
	s.whereParam.WriteString(" ORDER BY ")
	s.whereParam.WriteString(strings.Join(field, ","))
	s.whereParam.WriteString(" DESC")
	return s
}

func (s *NxjSession) Limit(limit int64) *NxjSession {
	s.limitValue = limit
	return s
}

func (s *NxjSession) Offset(offset int64) *NxjSession {
	s.offsetValue = offset
	return s
}

func (s *NxjSession) Count(count *int64) error {
	err := s.Aggregate("COUNT", "*", count)
	return err
}

// Aggregate 聚合函数
// funcName: 函数名称
// field: 字段
// result: 结果指针
func (s *NxjSession) Aggregate(funcName string, field string, result any) error {
	t := reflect.TypeOf(result)
	if t.Kind() != reflect.Ptr {
		return errors.New("result type must be a pointer")
	}
	query := fmt.Sprintf("SELECT %s(%s) FROM %s", funcName, field, s.tableName)
	var sb strings.Builder
	sb.WriteString(query)
	sb.WriteString(s.whereParam.String())
	sb.WriteString(";")
	s.db.logger.Info(sb.String())
	stmt, err := s.db.db.Prepare(sb.String())
	if err != nil {
		return err
	}
	row := stmt.QueryRow(s.whereValue...)
	err = row.Err()
	if err != nil {
		return err
	}
	err = row.Scan(result)
	return err
}

func (s *NxjSession) Exec(sqlStr string, values ...any) (int64, error) {
	s.db.logger.Info(sqlStr)
	var stmt *sql.Stmt
	var err error
	if s.beginTx {
		stmt, err = s.tx.Prepare(sqlStr)
	} else {
		stmt, err = s.db.db.Prepare(sqlStr)
	}
	if err != nil {
		return 0, err
	}
	r, err := stmt.Exec(values)
	if err != nil {
		return 0, err
	}
	if strings.Contains(strings.ToLower(sqlStr), "insert") {
		return r.LastInsertId()
	}
	return r.RowsAffected()
}

// QueryRow 查询单个结果
func (s *NxjSession) QueryRow(sqlStr string, data any, queryValues ...any) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr {
		panic(errors.New("data must be pointer"))
	}
	tVar := t.Elem()
	s.db.logger.Info(sqlStr)
	stmt, err := s.db.db.Prepare(sqlStr)
	if err != nil {
		return err
	}
	rows, err := stmt.Query(queryValues...)
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	values := make([]any, len(columns))
	fieldScan := make([]any, len(columns))
	for i := range fieldScan {
		fieldScan[i] = &values[i]
	}
	if rows.Next() {
		err = rows.Scan(fieldScan...)
		v := reflect.ValueOf(data)
		if err != nil {
			return err
		}
		for i := 0; i < tVar.NumField(); i++ {
			name := tVar.Field(i).Name
			tag := tVar.Field(i).Tag
			sqlTag := tag.Get("norm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(NameFormat(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]
					targetValue := reflect.ValueOf(target)
					fieldType := tVar.Field(i).Type
					result := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
					v.Elem().Field(i).Set(result)
				}
			}
		}
	}
	return nil
}

// QueryRows 查询所有结果
// sql: sql语句
// data: 结果集切片指针，必须为一个指向切片的指针
// queryValues: 查询参数
func (s *NxjSession) QueryRows(sql string, data any, queryValues ...any) error {
	t := reflect.TypeOf(data)
	if t.Kind() != reflect.Ptr || t.Elem().Kind() != reflect.Slice {
		return errors.New("data must be a pointer to a slice")
	}
	stmt, err := s.db.db.Prepare(sql)
	if err != nil {
		return err
	}
	rows, err := stmt.Query(queryValues...)
	if err != nil {
		return err
	}
	columns, err := rows.Columns()
	if err != nil {
		return err
	}
	t = t.Elem().Elem()
	v := reflect.ValueOf(data).Elem()
	for rows.Next() {
		newElem := reflect.New(t)
		values := make([]any, len(columns))
		fieldScan := make([]any, len(columns))
		for i := range fieldScan {
			fieldScan[i] = &values[i]
		}
		err = rows.Scan(fieldScan...)
		if err != nil {
			return err
		}
		for i := 0; i < t.NumField(); i++ {
			name := t.Field(i).Name
			tag := t.Field(i).Tag
			sqlTag := tag.Get("norm")
			if sqlTag == "" {
				sqlTag = strings.ToLower(NameFormat(name))
			} else {
				if strings.Contains(sqlTag, ",") {
					sqlTag = sqlTag[:strings.Index(sqlTag, ",")]
				}
			}
			for j, colName := range columns {
				if sqlTag == colName {
					target := values[j]
					targetValue := reflect.ValueOf(target)
					fieldType := t.Field(i).Type
					res := reflect.ValueOf(targetValue.Interface()).Convert(fieldType)
					newElem.Elem().FieldByName(name).Set(res)
				}
			}
		}
		newSlice := reflect.Append(v, newElem.Elem())
		v.Set(newSlice)
	}
	return nil
}

func (s *NxjSession) Begin() error {
	tx, err := s.db.db.Begin()
	if err != nil {
		return err
	}
	s.tx = tx
	s.beginTx = true
	return nil
}

func (s *NxjSession) Commit() error {
	err := s.tx.Commit()
	if err != nil {
		return err
	}
	s.beginTx = false
	return nil
}

func (s *NxjSession) Rollback() error {
	err := s.tx.Rollback()
	if err != nil {
		return err
	}
	s.beginTx = false
	return nil
}
