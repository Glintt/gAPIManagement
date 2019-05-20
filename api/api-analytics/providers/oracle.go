package providers

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"gAPIManagement/api/database"
	logsModels "gAPIManagement/api/logs/models"
	"strconv"
	"strings"
)

const LOGS_QUERY_ORACLE = `SELECT id,method,uri,request_body,host,user_agent,remote_addr,remote_ip,headers,query_args,date_time,response,elapsed_time,status_code,service_name,index_name,request_grouper_date FROM gapi_request_logs
where status_code >= 300
`
const ANALYTICS_QUERY_ORACLE = `  SELECT COUNT (*) total_requests,
MAX (elapsed_time) max_elapsed_time,
AVG (elapsed_time) avg_elapsed_time,
MIN (elapsed_time) min_elapsed_time,
(  SELECT LISTAGG (remote_addr || ' #||# ' || COUNT (remote_addr), ' #### ')
			  WITHIN GROUP (ORDER BY remote_addr)
			  AS remote_count
	 FROM gapi_request_logs b
	WHERE a.service_name = b.service_name
 GROUP BY remote_addr)
	AS remote_count,
(  SELECT LISTAGG (user_agent || ' #||# ' || COUNT (user_agent), ' #### ')
			  WITHIN GROUP (ORDER BY user_agent)
			  AS user_agent
	 FROM gapi_request_logs b
	WHERE a.service_name = b.service_name
 GROUP BY user_agent)
	AS user_agent_count,
	
(  SELECT LISTAGG (status_code || ' #||# ' || COUNT (status_code), ' #### ')
			  WITHIN GROUP (ORDER BY status_code)
			  AS status_code
	 FROM gapi_request_logs b
	WHERE a.service_name = b.service_name
 GROUP BY status_code)
	AS status_code_count,
service_name
FROM gapi_request_logs a
where index_name <> 'gapi-api-logs'
##WHERE_CLAUSE##
GROUP BY service_name`

func LogsOracle(apiEndpoint string) (string, int) {
	var err error
	var rows *sql.Rows

	db, err := database.ConnectToOracle(database.ORACLE_CONNECTION_STRING)
	if err != nil {
		return `{"error" : true, "msg": "` + err.Error() + ` "}`, 500
	}

	query := LOGS_QUERY_ORACLE

	if apiEndpoint != "" {
		//query = strings.Replace(query, "#WHERE_CLAUSE#", " where service_name = :serviceName", -1)
		query = query + " and service_name = :serviceName"
	}
	query = `SELECT * FROM
	(
		SELECT a.*, rownum r__
		FROM
		(
			` + query + `
			) a
			WHERE rownum < ((:page * 10) + 1 )
		)
		WHERE r__ >= (((:page-1) * 10) + 1)`

	if apiEndpoint != "" {
		rows, err = db.Query(query, apiEndpoint, 1)
	} else {
		rows, err = db.Query(query, 1)
	}
	if err != nil {
		return `{"error" : true, "msg": "` + err.Error() + ` "}`, 500
	}
	requestLogs := RowsToLogRequestModel(rows, true)

	defer rows.Close()
	database.CloseOracleConnection(db)

	var res []map[string]interface{}
	for _, v := range requestLogs {
		e := make(map[string]interface{})
		e["_source"] = v
		e["_id"] = v.Id
		res = append(res, e)
	}
	respString, _ := json.Marshal(res)
	return `{"hits":{"hits": ` + string(respString) + ` }}`, 200
}

func APIAnalyticsOracle(apiEndpoint string) (string, int) {
	var err error
	var rows *sql.Rows

	db, err := database.ConnectToOracle(database.ORACLE_CONNECTION_STRING)
	if err != nil {
		return `{"error" : true, "msg": "` + err.Error() + ` "}`, 500
	}

	query := ANALYTICS_QUERY_ORACLE
	if apiEndpoint != "" {
		query = strings.Replace(query, "##WHERE_CLAUSE##", " and service_name = :serviceName", -1)
		rows, err = db.Query(query, apiEndpoint)
	} else {
		query = strings.Replace(query, "##WHERE_CLAUSE##", "", -1)
		rows, err = db.Query(query)
	}

	apiAnalytics := RowsToApiAnalyticsModel(rows, true)

	defer rows.Close()
	database.CloseOracleConnection(db)

	respString, _ := json.Marshal(apiAnalytics)
	return `{"aggregations":{"api": {"buckets":` + string(respString) + ` }}}`, 200
}

type ApiAnalytics struct {
	Key            string `json:"key"`
	MaxElapsedTime MaxElapsedTimeStruct
	MinElapsedTime MinElapsedTimeStruct
	AvgElapsedTime AvgElapsedTimeStruct
	TotalRequests  int `json:"doc_count"`
	RemoteAddr     RemoteAddrStruct
	UserAgent      UserAgentStruct
	StatusCode     StatusCodeStruct
}

type MinElapsedTimeStruct struct {
	Value float32 `json:"value"`
}
type MaxElapsedTimeStruct struct {
	Value float32 `json:"value"`
}
type AvgElapsedTimeStruct struct {
	Value float32 `json:"value"`
}
type BucketStruct struct {
	Key      string `json:"key"`
	DocCount int    `json:"doc_count"`
}
type UserAgentStruct struct {
	Buckets []BucketStruct `json:"buckets"`
}
type RemoteAddrStruct struct {
	Buckets []BucketStruct `json:"buckets"`
}
type StatusCodeStruct struct {
	Buckets []BucketStruct `json:"buckets"`
}

func RowsToApiAnalyticsModel(rows *sql.Rows, containsPagination bool) []ApiAnalytics {
	var analytics []ApiAnalytics
	for rows.Next() {
		var a ApiAnalytics

		var remoteCount, userAgentCount, statusCodeCount string

		rows.Scan(&a.TotalRequests,
			&a.MaxElapsedTime.Value,
			&a.AvgElapsedTime.Value,
			&a.MinElapsedTime.Value,
			&remoteCount,
			&userAgentCount,
			&statusCodeCount,
			&a.Key,
		)

		a.RemoteAddr.Buckets = []BucketStruct{}
		a.UserAgent.Buckets = []BucketStruct{}
		a.StatusCode.Buckets = []BucketStruct{}

		remotesList := strings.Split(remoteCount, " #### ")
		for _, r := range remotesList {
			remCount := strings.Split(r, " #||# ")
			if len(remCount) != 2 {
				continue
			}
			count, _ := strconv.Atoi(remCount[1])
			a.RemoteAddr.Buckets = append(a.RemoteAddr.Buckets, BucketStruct{
				Key:      remCount[0],
				DocCount: count,
			})
		}

		userAgentList := strings.Split(userAgentCount, " #### ")
		for _, r := range userAgentList {
			userAgentCount := strings.Split(r, " #||# ")
			if len(userAgentCount) != 2 {
				continue
			}

			count, _ := strconv.Atoi(userAgentCount[1])
			a.UserAgent.Buckets = append(a.UserAgent.Buckets, BucketStruct{
				Key:      userAgentCount[0],
				DocCount: count,
			})
		}
		statusCodeList := strings.Split(statusCodeCount, " #### ")
		for _, r := range statusCodeList {
			statusCodeCount := strings.Split(r, " #||# ")
			fmt.Println(statusCodeCount)
			if len(statusCodeCount) != 2 {
				continue
			}
			count, _ := strconv.Atoi(statusCodeCount[1])
			a.StatusCode.Buckets = append(a.StatusCode.Buckets, BucketStruct{
				Key:      statusCodeCount[0],
				DocCount: count,
			})
		}

		analytics = append(analytics, a)
	}

	return analytics
}

func RowsToLogRequestModel(rows *sql.Rows, containsPagination bool) []logsModels.RequestLogging {
	var logs []logsModels.RequestLogging
	for rows.Next() {
		var s logsModels.RequestLogging
		var a int
		var currentDate string
		if containsPagination {
			rows.Scan(
				&s.Id,
				&s.Method,
				&s.Uri,
				&s.RequestBody,
				&s.Host,
				&s.UserAgent,
				&s.RemoteAddr,
				&s.RemoteIp,
				&s.Headers,
				&s.QueryArgs,
				&s.DateTime,
				&s.Response,
				&s.ElapsedTime,
				&s.StatusCode,
				&s.ServiceName,
				&s.IndexName,
				&currentDate,
				&a,
			)
		} else {
			rows.Scan(
				&s.Id, &s.Method,
				&s.Uri,
				&s.RequestBody,
				&s.Host,
				&s.UserAgent,
				&s.RemoteAddr,
				&s.RemoteIp,
				&s.Headers,
				&s.QueryArgs,
				&s.DateTime,
				&s.Response,
				&s.ElapsedTime,
				&s.StatusCode,
				&s.ServiceName,
				&s.IndexName,
				&currentDate,
			)
		}

		logs = append(logs, s)
	}

	defer rows.Close()

	return logs
}
