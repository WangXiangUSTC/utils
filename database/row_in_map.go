package database

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/zr-hebo/utils/container"
)

const (
	dbTypeMysql       = "mysql"
	overflowIn64Value = 9223372036854775808
)

// Host  主机
type Host struct {
	IP     string `json:"ip"`
	Domain string `json:"domain"`
	Port   int    `json:"port"`
}

// UnanimityHost  id标示的主机
type UnanimityHost struct {
	Host string `json:"host"`
	Port int    `json:"port"`
}

func (uh *UnanimityHost) String() string {
	return fmt.Sprintf("%s:%d", uh.Host, uh.Port)
}

// UnanimityHostWithDomains   带域名的id标示的主机
type UnanimityHostWithDomains struct {
	UnanimityHost
	IP      string   `json:"ip"`
	Domains []string `json:"domains"`
}

// Field 字段
type Field struct {
	Name string
	Type string
}

// FieldType Common type include "STRING", "FLOAT", "INT", "BOOL"
func (f *Field) FieldType() string {
	return f.Type
}

// QueryRowInMap 查询单行数据
type QueryRowInMap struct {
	Fields []Field
	Record map[string]interface{}
}

// QueryRowInOrderedMap 查询单行数据
type QueryRowInOrderedMap struct {
	Fields []Field
	Record *container.OrderedMap
}

// QueryRow 查询单行数据
type QueryRow struct {
	Fields []Field
	Record []interface{}
}

// QueryRowsInMap 查询多行数据
type QueryRowsInMap struct {
	Fields  []Field
	Records []map[string]interface{}
}

// QueryRowsInOrderedMap 查询多行数据
type QueryRowsInOrderedMap struct {
	Fields  []Field
	Records []*container.OrderedMap
}

// QueryRows 查询多行数据
type QueryRows struct {
	Fields  []Field
	Records [][]interface{}
}

func newQueryRowInMap() *QueryRowInMap {
	queryRow := new(QueryRowInMap)
	queryRow.Fields = make([]Field, 0)
	queryRow.Record = make(map[string]interface{})
	return queryRow
}

func newQueryRowInOrderedMap() *QueryRowInOrderedMap {
	queryRow := new(QueryRowInOrderedMap)
	queryRow.Fields = make([]Field, 0)
	queryRow.Record = container.NewOrderedMap()
	return queryRow
}

func newQueryRow() *QueryRow {
	queryRow := new(QueryRow)
	queryRow.Fields = make([]Field, 0)
	queryRow.Record = make([]interface{}, 0)
	return queryRow
}

func newQueryRowsInMap() *QueryRowsInMap {
	queryRows := new(QueryRowsInMap)
	queryRows.Fields = make([]Field, 0)
	queryRows.Records = make([]map[string]interface{}, 0)
	return queryRows
}

func newQueryRowsInOrderedMap() *QueryRowsInOrderedMap {
	queryRows := new(QueryRowsInOrderedMap)
	queryRows.Fields = make([]Field, 0)
	queryRows.Records = make([]*container.OrderedMap, 0, 8)
	return queryRows
}

func newQueryRows() *QueryRows {
	queryRows := new(QueryRows)
	queryRows.Fields = make([]Field, 0)
	queryRows.Records = make([][]interface{}, 0, 8)
	return queryRows
}

// MySQL Mysql主机实例
type MySQL struct {
	Host
	UserName         string
	Passwd           string
	DatabaseType     string
	DBName           string
	MultiStatements  bool
	MaxLifetime      int
	QueryTimeout     int
	UseSSL           bool
	MaxAllowedPacket int
	maxIdleConns     int
	maxOpenConns     int
	// https://github.com/go-sql-driver/mysql#interpolateparams
	InterpolateParams bool

	connectionLock sync.Mutex
	rawDB          *sql.DB
}

// NewMySQL 创建MySQL数据库
func NewMySQL(
	ip string, port int, userName, passwd, dbName string) (mysql *MySQL, err error) {
	mysql = new(MySQL)
	mysql.DatabaseType = dbTypeMysql
	mysql.QueryTimeout = 30
	mysql.IP = ip
	mysql.Port = port
	mysql.UserName = userName
	mysql.Passwd = passwd
	mysql.DBName = dbName

	return
}

// NewMySQLWithTimeout 创建MySQL数据库
func NewMySQLWithTimeout(
	ip string, port int, userName, passwd, dbName string, timeout int) (mysql *MySQL, err error) {
	mysql = new(MySQL)
	mysql.DatabaseType = dbTypeMysql
	mysql.QueryTimeout = timeout
	mysql.IP = ip
	mysql.Port = port
	mysql.UserName = userName
	mysql.Passwd = passwd
	mysql.DBName = dbName

	return
}

// SetConnMaxLifetime 设置连接超时时间
func (m *MySQL) SetConnMaxLifetime(lifetime int) {
	m.MaxLifetime = lifetime
	return
}

// SetMaxIdleConns 设置最大空闲连接
func (m *MySQL) SetMaxIdleConns(n int) {
	m.maxIdleConns = n
}

// SetMaxOpenConns 设置最大连接数
func (m *MySQL) SetMaxOpenConns(n int) {
	m.maxOpenConns = n
}

// Close 关闭数据库连接
func (m *MySQL) Close() (err error) {
	if m.rawDB != nil {
		err = m.rawDB.Close()
		m.rawDB = nil
	}
	return
}

// RawDB 获取数据库连接
func (m *MySQL) RawDB() (db *sql.DB, err error) {
	m.connectionLock.Lock()
	defer m.connectionLock.Unlock()

	if m.rawDB == nil {
		conn, err := sql.Open(m.DatabaseType, m.fillConnStr())
		if err != nil {
			return nil, err
		}

		if m.MaxLifetime != 0 {
			conn.SetConnMaxLifetime(time.Second * time.Duration(m.MaxLifetime))
		}
		if m.maxOpenConns != 0 {
			conn.SetMaxOpenConns(m.maxOpenConns)
		}
		if m.maxIdleConns != 0 {
			conn.SetMaxIdleConns(m.maxIdleConns)
		}
		m.rawDB = conn
	}

	db = m.rawDB
	return
}

// OpenSession 获取数据库连接
func (m *MySQL) OpenSession(ctx context.Context) (session *sql.Conn, err error) {
	rawDB, err := m.RawDB()
	if err != nil {
		return
	}

	session, err = rawDB.Conn(ctx)
	return
}

// QueryRowsInMap 执行 MySQL Query语句，返回多条数据
func (m *MySQL) QueryRowsInMap(querySQL string, args ...interface{}) (queryRows *QueryRowsInMap, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.QueryTimeout)*time.Second)
	defer cancel()
	return m.QueryRowsInMapWithContext(ctx, querySQL, args...)
}

// QueryRowsInOrderedMap 执行 MySQL Query语句，返回多条数据
func (m *MySQL) QueryRowsInOrderedMap(querySQL string, args ...interface{}) (
	queryRows *QueryRowsInOrderedMap, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.QueryTimeout)*time.Second)
	defer cancel()
	return m.QueryRowsInOrderedMapWithContext(ctx, querySQL, args...)
}

// QueryRowsInMapWithContext 执行 MySQL Query语句，返回多条数据
func (m *MySQL) QueryRowsInMapWithContext(ctx context.Context, querySQL string, args ...interface{}) (
	queryRows *QueryRowsInMap, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("query rows on %s:%d failed <-- %s", m.IP, m.Port, err.Error())
		}
	}()

	session, err := m.OpenSession(ctx)
	defer func() {
		if session != nil {
			_ = session.Close()
		}
	}()
	if err != nil {
		return nil, err
	}

	rawRows, err := session.QueryContext(ctx, querySQL, args...)
	defer func() {
		if rawRows != nil {
			_ = rawRows.Close()
		}
	}()
	if err != nil {
		return
	}

	colTypes, err := rawRows.ColumnTypes()
	if err != nil {
		return
	}

	fields := make([]Field, 0, len(colTypes))
	for _, colType := range colTypes {
		colType.ScanType()
		fields = append(fields, Field{Name: colType.Name(), Type: getDataType(colType.DatabaseTypeName())})
	}

	queryRows = newQueryRowsInMap()
	queryRows.Fields = fields
	for rawRows.Next() {
		receiver := createReceivers(fields)
		err = rawRows.Scan(receiver...)
		if err != nil {
			err = fmt.Errorf("scan rows failed <-- %s", err.Error())
			return
		}

		queryRows.Records = append(queryRows.Records, getRecordInMapFromReceiver(receiver, fields))
	}

	err = rawRows.Err()
	return
}

// QueryRowsInOrderedMapWithContext 执行 MySQL Query语句，返回多条数据
func (m *MySQL) QueryRowsInOrderedMapWithContext(ctx context.Context, querySQL string, args ...interface{}) (
	queryRows *QueryRowsInOrderedMap, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("query rows on %s:%d failed <-- %s", m.IP, m.Port, err.Error())
		}
	}()

	session, err := m.OpenSession(ctx)
	defer func() {
		if session != nil {
			_ = session.Close()
		}
	}()
	if err != nil {
		return nil, err
	}

	rawRows, err := session.QueryContext(ctx, querySQL, args...)
	defer func() {
		if rawRows != nil {
			_ = rawRows.Close()
		}
	}()
	if err != nil {
		return
	}

	colTypes, err := rawRows.ColumnTypes()
	if err != nil {
		return
	}

	fields := make([]Field, 0, len(colTypes))
	for _, colType := range colTypes {
		colType.ScanType()
		fields = append(fields, Field{Name: colType.Name(), Type: getDataType(colType.DatabaseTypeName())})
	}

	queryRows = newQueryRowsInOrderedMap()
	queryRows.Fields = fields
	for rawRows.Next() {
		receiver := createReceivers(fields)
		err = rawRows.Scan(receiver...)
		if err != nil {
			err = fmt.Errorf("scan rows failed <-- %s", err.Error())
			return
		}

		queryRows.Records = append(queryRows.Records, getRecordInOrderedMapFromReceiver(receiver, fields))
	}

	err = rawRows.Err()
	return
}

// QueryRowsWithMapInTx 执行 MySQL Query语句，返回多条数据
func QueryRowsWithMapInTx(ctx context.Context, tx *sql.Tx, querySQL string, args ...interface{}) (
	queryRows *QueryRowsInMap, err error) {
	rawRows, err := tx.QueryContext(ctx, querySQL, args...)
	// rawRows, err := db.Query(stmt)
	defer func() {
		if rawRows != nil {
			_ = rawRows.Close()
		}
	}()
	if err != nil {
		return
	}

	colTypes, err := rawRows.ColumnTypes()
	if err != nil {
		return
	}

	fields := make([]Field, 0, len(colTypes))
	for _, colType := range colTypes {
		fields = append(fields, Field{Name: colType.Name(), Type: getDataType(colType.DatabaseTypeName())})
	}

	queryRows = newQueryRowsInMap()
	queryRows.Fields = fields
	for rawRows.Next() {
		receiver := createReceivers(fields)
		err = rawRows.Scan(receiver...)
		if err != nil {
			err = fmt.Errorf("scan rows failed <-- %s", err.Error())
			return
		}

		queryRows.Records = append(queryRows.Records, getRecordInMapFromReceiver(receiver, fields))
	}

	err = rawRows.Err()
	return
}

// QueryRowsWithOrderedMapInTx 执行 MySQL Query语句，返回多条数据
func QueryRowsWithOrderedMapInTx(ctx context.Context, tx *sql.Tx, querySQL string, args ...interface{}) (
	queryRows *QueryRowsInOrderedMap, err error) {
	rawRows, err := tx.QueryContext(ctx, querySQL, args...)
	// rawRows, err := db.Query(stmt)
	defer func() {
		if rawRows != nil {
			_ = rawRows.Close()
		}
	}()
	if err != nil {
		return
	}

	colTypes, err := rawRows.ColumnTypes()
	if err != nil {
		return
	}

	fields := make([]Field, 0, len(colTypes))
	for _, colType := range colTypes {
		fields = append(fields, Field{Name: colType.Name(), Type: getDataType(colType.DatabaseTypeName())})
	}

	queryRows = newQueryRowsInOrderedMap()
	queryRows.Fields = fields
	for rawRows.Next() {
		receiver := createReceivers(fields)
		err = rawRows.Scan(receiver...)
		if err != nil {
			err = fmt.Errorf("scan rows failed <-- %s", err.Error())
			return
		}

		queryRows.Records = append(queryRows.Records, getRecordInOrderedMapFromReceiver(receiver, fields))
	}

	err = rawRows.Err()
	return
}

// QueryRowWithMapInTx 执行 MySQL Query语句，返回１条或０条数据
func QueryRowWithMapInTx(ctx context.Context, tx *sql.Tx, stmt string, args ...interface{}) (
	row *QueryRowInMap, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("query row failed <-- %s", err.Error())
		}
	}()

	queryRows, err := QueryRowsWithMapInTx(ctx, tx, stmt, args...)
	if err != nil || queryRows == nil {
		return
	}

	if len(queryRows.Records) < 1 {
		return
	}

	row = newQueryRowInMap()
	row.Fields = queryRows.Fields
	row.Record = queryRows.Records[0]

	return
}

// QueryRowWithOrderedMapInTx 执行 MySQL Query语句，返回１条或０条数据
func QueryRowWithOrderedMapInTx(ctx context.Context, tx *sql.Tx, stmt string, args ...interface{}) (
	row *QueryRowInOrderedMap, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("query row failed <-- %s", err.Error())
		}
	}()

	queryRows, err := QueryRowsWithOrderedMapInTx(ctx, tx, stmt, args...)
	if err != nil || queryRows == nil {
		return
	}

	if len(queryRows.Records) < 1 {
		return
	}

	row = newQueryRowInOrderedMap()
	row.Fields = queryRows.Fields
	row.Record = queryRows.Records[0]
	return
}

// QueryRowInMap 执行 MySQL Query语句，返回１条或０条数据
func (m *MySQL) QueryRowInMap(querySQL string, args ...interface{}) (row *QueryRowInMap, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("query row failed <-- %s", err.Error())
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.QueryTimeout)*time.Second)
	defer cancel()
	return m.QueryRowInMapWithContext(ctx, querySQL, args...)
}

// QueryRowInOrderedMap 执行 MySQL Query语句，返回１条或０条数据
func (m *MySQL) QueryRowInOrderedMap(querySQL string, args ...interface{}) (row *QueryRowInOrderedMap, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("query row failed <-- %s", err.Error())
		}
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(m.QueryTimeout)*time.Second)
	defer cancel()
	return m.QueryRowInOrderedMapWithContext(ctx, querySQL, args...)
}

// QueryRowInMapWithContext 执行 MySQL Query语句，返回１条或０条数据
func (m *MySQL) QueryRowInMapWithContext(ctx context.Context, querySQL string, args ...interface{}) (
	row *QueryRowInMap, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("query row failed <-- %s", err.Error())
		}
	}()

	queryRows, err := m.QueryRowsInMapWithContext(ctx, querySQL, args...)
	if err != nil || queryRows == nil {
		return
	}

	if len(queryRows.Records) < 1 {
		return
	}

	row = newQueryRowInMap()
	row.Fields = queryRows.Fields
	row.Record = queryRows.Records[0]

	return
}

// QueryRowInOrderedMapWithContext 执行 MySQL Query语句，返回１条或０条数据
func (m *MySQL) QueryRowInOrderedMapWithContext(ctx context.Context, querySQL string, args ...interface{}) (
	row *QueryRowInOrderedMap, err error) {
	defer func() {
		if err != nil {
			err = fmt.Errorf("query row failed <-- %s", err.Error())
		}
	}()

	resp, err := m.QueryRowsInOrderedMapWithContext(ctx, querySQL, args...)
	if err != nil || resp == nil {
		return
	}

	if len(resp.Records) < 1 {
		return
	}

	row = newQueryRowInOrderedMap()
	row.Fields = resp.Fields
	row.Record = resp.Records[0]
	return
}

func (m *MySQL) fetchRowsInMapAsync(ctx context.Context,
	recordChan chan map[string]interface{}, fields []Field, querySQL string, args ...interface{}) {
	var err error
	defer func() {
		if err != nil {
			panic(err.Error())
		}
		close(recordChan)
	}()

	session, err := m.OpenSession(ctx)
	defer func() {
		if session != nil {
			_ = session.Close()
		}
	}()
	if err != nil {
		return
	}

	rawRows, err := session.QueryContext(ctx, querySQL, args...)
	defer func() {
		if rawRows != nil {
			_ = rawRows.Close()
		}
	}()
	if err != nil {
		return
	}

	for {
		select {
		case <-ctx.Done():
			fmt.Printf("async query context canceled <-- %s\n", ctx.Err().Error())
			return

		default:
			if rawRows.Next() {
				receiver := createReceivers(fields)
				err = rawRows.Scan(receiver...)
				if err != nil {
					panic(fmt.Sprintf("scan rows failed <-- %s", err.Error()))
				}

				recordChan <- getRecordInMapFromReceiver(receiver, fields)

			} else {
				err = rawRows.Err()
				if err != nil {
					panic(fmt.Sprintf("async query failed <-- %s", err.Error()))
				}
				return
			}
		}
	}
}

func createReceivers(fields []Field) (receivers []interface{}) {
	receivers = make([]interface{}, 0, len(fields))
	for _, field := range fields {
		switch field.Type {
		case "string":
			{
				var val sql.NullString
				receivers = append(receivers, &val)
			}
		case "int32":
			{
				var val sql.NullInt64
				receivers = append(receivers, &val)
			}
		case "int64":
			{
				var val sql.NullString
				receivers = append(receivers, &val)
			}
		case "float32":
			{
				var val sql.NullFloat64
				receivers = append(receivers, &val)
			}
		case "float64":
			{
				var val sql.NullFloat64
				receivers = append(receivers, &val)
			}
		case "bool":
			{
				var val sql.NullBool
				receivers = append(receivers, &val)
			}
		case "binary":
			{
				var val sql.RawBytes
				receivers = append(receivers, &val)
			}
		case "blob":
			{
				var val sql.RawBytes
				receivers = append(receivers, &val)
			}
		default:
			var val sql.NullString
			receivers = append(receivers, &val)
		}
	}

	return
}

func getRecordInMapFromReceiver(receiver []interface{}, fields []Field) (record map[string]interface{}) {
	record = make(map[string]interface{}, len(fields))
	for idx := 0; idx < len(fields); idx++ {
		field := fields[idx]
		value := receiver[idx]
		switch field.Type {
		case "string":
			{
				nullVal := value.(*sql.NullString)
				record[field.Name] = nil
				if nullVal.Valid {
					record[field.Name] = nullVal.String
				}
			}
		case "int32":
			{
				nullVal := value.(*sql.NullInt64)
				record[field.Name] = nil
				if nullVal.Valid {
					record[field.Name] = nullVal.Int64
				}
			}
		case "int64":
			{
				nullVal := value.(*sql.NullString)
				record[field.Name] = nil
				if nullVal.Valid {
					if nullVal.String[0] == '-' {
						intVal, err := strconv.ParseInt(nullVal.String, 10, 64)
						if err != nil {
							panic(fmt.Sprintf("parse int64 value from '%s' failed <-- %s",
								nullVal.String, err.Error()))
						}
						record[field.Name] = intVal

					} else {
						uintVal, err := strconv.ParseUint(nullVal.String, 10, 64)
						if err != nil {
							panic(fmt.Sprintf("parse uint64 value from '%s' failed <-- %s",
								nullVal.String, err.Error()))
						}
						if uintVal < overflowIn64Value {
							record[field.Name] = int64(uintVal)
						} else {
							record[field.Name] = uintVal
						}
					}
				}
			}
		case "float64":
			{
				nullVal := value.(*sql.NullFloat64)
				record[field.Name] = nil
				if nullVal.Valid {
					record[field.Name] = nullVal.Float64
				}
			}
		case "float32":
			{
				nullVal := value.(*sql.NullFloat64)
				record[field.Name] = nil
				if nullVal.Valid {
					record[field.Name] = float32(nullVal.Float64)
				}
			}
		case "bool":
			{
				nullVal := value.(*sql.NullBool)
				record[field.Name] = nil
				if nullVal.Valid {
					record[field.Name] = nullVal.Bool
				}
			}
		case "blob":
			{
				rawVal := value.(*sql.RawBytes)
				record[field.Name] = nil
				if rawVal != nil && *rawVal != nil {
					val := make([]byte, len(*rawVal))
					copy(val, *rawVal)
					record[field.Name] = val
				}
			}
		default:
			{
				nullVal := value.(*sql.NullString)
				record[field.Name] = nil
				if nullVal.Valid {
					record[field.Name] = nullVal.String
				}
			}
		}
	}
	return
}

func getRecordInOrderedMapFromReceiver(receiver []interface{}, fields []Field) (record *container.OrderedMap) {
	record = container.NewOrderedMapWithSize(len(fields))
	for idx := 0; idx < len(fields); idx++ {
		field := fields[idx]
		value := receiver[idx]
		switch field.Type {
		case "string":
			{
				nullVal := value.(*sql.NullString)
				record.Set(field.Name, nil)
				if nullVal.Valid {
					record.Set(field.Name, nullVal.String)
				}
			}
		case "int32":
			{
				nullVal := value.(*sql.NullInt64)
				record.Set(field.Name, nil)
				if nullVal.Valid {
					record.Set(field.Name, nullVal.Int64)
				}
			}
		case "int64":
			{
				nullVal := value.(*sql.NullString)
				record.Set(field.Name, nil)
				if nullVal.Valid {
					if nullVal.String[0] == '-' {
						intVal, err := strconv.ParseInt(nullVal.String, 10, 64)
						if err != nil {
							panic(fmt.Sprintf("parse int64 value from '%s' failed <-- %s",
								nullVal.String, err.Error()))
						}
						record.Set(field.Name, intVal)

					} else {
						uintVal, err := strconv.ParseUint(nullVal.String, 10, 64)
						if err != nil {
							panic(fmt.Sprintf("parse uint64 value from '%s' failed <-- %s",
								nullVal.String, err.Error()))
						}
						if uintVal < overflowIn64Value {
							record.Set(field.Name, int64(uintVal))

						} else {
							record.Set(field.Name, uintVal)

						}
					}
				}
			}
		case "float64":
			{
				nullVal := value.(*sql.NullFloat64)
				record.Set(field.Name, nil)
				if nullVal.Valid {
					record.Set(field.Name, nullVal.Float64)
				}
			}
		case "float32":
			{
				nullVal := value.(*sql.NullFloat64)
				record.Set(field.Name, nil)
				if nullVal.Valid {
					record.Set(field.Name, nullVal.Float64)
				}
			}
		case "bool":
			{
				nullVal := value.(*sql.NullBool)
				record.Set(field.Name, nil)
				if nullVal.Valid {
					record.Set(field.Name, nullVal.Bool)
				}
			}
		case "blob":
			{
				rawVal := value.(*sql.RawBytes)
				record.Set(field.Name, nil)
				if rawVal != nil && *rawVal != nil {
					val := make([]byte, len(*rawVal))
					copy(val, *rawVal)
					record.Set(field.Name, val)
				}
			}
		default:
			{
				nullVal := value.(*sql.NullString)
				record.Set(field.Name, nil)
				if nullVal.Valid {
					record.Set(field.Name, nullVal.String)
				}
			}
		}
	}
	return
}

var columnTypeDict = map[string]string{
	"CHAR":       "string",
	"VARCHAR":    "string",
	"NVARCHAR":   "string",
	"DATE":       "string",
	"TIME":       "string",
	"YEAR":       "string",
	"DATETIME":   "string",
	"TIMESTAMP":  "string",
	"DECIMAL":    "string",
	"FLOAT":      "float32",
	"DOUBLE":     "float64",
	"BOOL":       "bool",
	"TINYINT":    "int32",
	"SMALLINT":   "int32",
	"MEDIUMINT":  "int32",
	"INT":        "int32",
	"INTEGER":    "int32",
	"BIGINT":     "int64",
	"BINARY":     "binary",
	"VARBINARY":  "blob",
	"BLOB":       "blob",
	"TINYBLOB":   "blob",
	"MEDIUMBLOB": "blob",
	"LONGBLOB":   "blob",
	"TEXT":       "string",
	"TINYTEXT":   "string",
	"MEDIUMTEXT": "string",
	"LONGTEXT":   "string",
}

func getDataType(dbColType string) (colType string) {
	colType, ok := columnTypeDict[dbColType]
	if ok {
		return
	}

	colType = "string"
	return
}

func (m *MySQL) fillConnStr() string {
	dbServerInfoStr := fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?multiStatements=%v&interpolateParams=%v&maxAllowedPacket=%d",
		m.UserName, m.Passwd, m.IP, m.Port, m.DBName, m.MultiStatements, m.InterpolateParams, m.MaxAllowedPacket)
	if m.QueryTimeout > 0 {
		dbServerInfoStr = fmt.Sprintf("%s&timeout=3s&readTimeout=%ds&writeTimeout=%ds",
			dbServerInfoStr, m.QueryTimeout, m.QueryTimeout)
	}
	if m.UseSSL {
		dbServerInfoStr = fmt.Sprintf("%s&tls=skip-verify", dbServerInfoStr)
	}

	return dbServerInfoStr
}

// Exec 执行 MySQL dml语句，返回执行结果
func (m *MySQL) Exec(query string, args ...interface{}) (sql.Result, error) {
	session, err := m.OpenSession(context.Background())
	if session != nil {
		defer session.Close()
	}
	if err != nil {
		return nil, err
	}
	return session.ExecContext(context.Background(), query, args...)
}

// ExecContext 执行MySQL dml语句，返回执行结果
func (m *MySQL) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	session, err := m.OpenSession(ctx)
	defer func() {
		if session != nil {
			_ = session.Close()
		}
	}()
	if err != nil {
		return nil, err
	}

	return session.ExecContext(ctx, query, args...)
}
