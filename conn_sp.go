package freetds

import (
	"errors"
	"fmt"
	"unsafe"
)

/*
 #cgo LDFLAGS: -lsybdb
 #include <stdio.h>
 #include <stdlib.h>
 #include <string.h>
 #include <assert.h>
 #include <errno.h>
 #include <unistd.h>
 #include <libgen.h>

 #include <sybfront.h>
 #include <sybdb.h>
*/
import "C"

//Execute stored procedure by name and list of params.
//
//Example:
//  conn.ExecSp("sp_help", "authors")
func (conn *Conn) ExecSp(spName string, params ...interface{}) (*SpResult, error) {
	conn.clearMessages()
	if C.dbrpcinit(conn.dbproc, C.CString(spName), 0) == C.FAIL {
		return nil, conn.raiseError("dbrpcinit failed")
	}
	//input params
	spParams, err := conn.getSpParams(spName)
	if err != nil {
		return nil, err
	}
	for i, spParam := range spParams {
		//get datavalue for the suplied stored procedure parametar
		var datavalue *C.BYTE
		datalen := C.DBINT(0)
		if i < len(params) {
			param := params[i]
			if param != nil {
				data, err := typeToSqlBuf(int(spParam.UserTypeId), param)
				if err != nil {
					return nil, err
				}
				if len(data) > 0 {
					datalen = C.DBINT(len(data))
					datavalue = (*C.BYTE)(unsafe.Pointer(&data[0]))
					//fmt.Printf("dbrpcparam %s: [%v] '%s'\n", spParam.Name, data, data)
				}
			}
		}
		//set parametar valus, call dbrpcparam
		if i < len(params) || spParam.IsOutput {
			maxOutputSize := C.DBINT(0)
			status := C.BYTE(0)
			if spParam.IsOutput {
				status = C.DBRPCRETURN
				maxOutputSize = C.DBINT(spParam.MaxLength)
			}
			paramname := C.CString(spParam.Name)
			defer C.free(unsafe.Pointer(paramname))
			if C.dbrpcparam(conn.dbproc, paramname, status,
				C.int(spParam.UserTypeId), maxOutputSize, datalen, datavalue) == C.FAIL {
				return nil, errors.New("dbrpcparam failed")
			}
		}
	}
	//execute
	if C.dbrpcsend(conn.dbproc) == C.FAIL {
		return nil, conn.raiseError("dbrpcsend failed")
	}
	//results
	result := NewSpResult()
	result.results, err = conn.fetchResults()
	if err != nil {
		return nil, conn.raise(err)
	}
	//return status
	if C.dbhasretstat(conn.dbproc) == C.TRUE {
		result.status = int(C.dbretstatus(conn.dbproc))
	}
	//read output params
	numOutParams := int(C.dbnumrets(conn.dbproc))
	result.outputParams = make([]*SpOutputParam, numOutParams)
	for i := 1; i <= numOutParams; i++ {
		j := C.int(i)
		len := C.dbretlen(conn.dbproc, j)
		name := C.GoString(C.dbretname(conn.dbproc, j))
		typ := int(C.dbrettype(conn.dbproc, j))
		data := C.GoBytes(unsafe.Pointer(C.dbretdata(conn.dbproc, j)), len)
		value := sqlBufToType(typ, data)
		param := &SpOutputParam{Name: name, Value: value}
		result.outputParams[i-1] = param
	}

	return result, nil
}

func (conn *Conn) raise(err error) error {
	if len(conn.Error) != 0 {
		return errors.New(fmt.Sprintf("%s\n%s", conn.Error, conn.Message))
	} else {
		return err
	}
}

func (conn *Conn) raiseError(errMsg string) error {
	return conn.raise(errors.New(errMsg))
}

// func toRpcParam(datatype int, value interface{}) (datalen C.DBINT, datavalue *C.BYTE, err error) {
// 	data, err := typeToSqlBuf(datatype, value)
// 	if err != nil {
// 		return
// 	}
// 	datalen = C.DBINT(len(data))
// 	if len(data) > 0 {
// 		datavalue = (*C.BYTE)(unsafe.Pointer(&data[0]))
// 	}
// 	//fmt.Printf("\ndatavalue: %v, datalen: %v, data: %v %s\n", datavalue, datalen, data, data)
// 	return
// }

//Stored procedure parameter definition
type spParam struct {
	Name        string
	ParameterId int32
	UserTypeId  int32
	IsOutput    bool
	MaxLength   int16
	Precision   uint8
	Scale       uint8
}

//Read stored procedure parameters.
//Will cache params in connection or pool and reuse it.
func (conn *Conn) getSpParams(spName string) ([]*spParam, error) {
	if spParams, ok := conn.spParamsCache[spName]; ok {
		return spParams, nil
	}

	sql := fmt.Sprintf(`
select name, parameter_id, user_type_id, is_output, max_length, precision, scale
from sys.all_parameters
where object_id =  (select object_id from sys.all_objects where object_id = object_id('%s'))
order by parameter_id
`, spName)
	results, err := conn.exec(sql)
	if err != nil {
		return nil, err
	}
	r := results[0]
	spParams := make([]*spParam, len(r.Rows))
	for i := 0; r.Next(); i++ {
		p := &spParam{}
		err := r.Scan(&p.Name, &p.ParameterId, &p.UserTypeId, &p.IsOutput, &p.MaxLength, &p.Precision, &p.Scale)
		//fixme: mapping uniqueidentifier to string
		if p.UserTypeId == 36 { 
			p.UserTypeId = 39
		}
		if err != nil {
			return nil, err
		}
		spParams[i] = p
	}

	conn.spParamsCache[spName] = spParams
	return spParams, nil
}
