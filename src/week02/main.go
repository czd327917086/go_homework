package main

import (
	"database/sql"
	"fmt"
	"net/http"
	"strings"

	"github.com/pkg/errors"

	_ "github.com/go-sql-driver/mysql"
)

type UserHandler struct {
	dao *Dao
}

func (h *UserHandler) DisplayUserName(rsp http.ResponseWriter, r *http.Request) {
	r.ParseForm()
	userAccount := r.FormValue("userAccount")
	userAccount = strings.TrimSpace(userAccount)
	if userAccount == "" {
		rsp.Write([]byte("read arg userAccount is blank"))
		return
	}
	userName, err := h.dao.GetUserInfo(userAccount)
	if err == nil {
		rsp.Write([]byte(userName))
		return
	}
	if errors.Is(err, sql.ErrNoRows) {
		_, _ = rsp.Write([]byte(fmt.Sprintf("user is not exists with account '%s'", userAccount)))
		return
	}
	//err = errors.Cause(err)
	rsp.Write([]byte(err.Error()))
}

type Dao struct {
	db *sql.DB
}

func (d *Dao) GetUserInfo(account string) (name string, err error) {
	query := "select username from t_user where account=?"
	e := d.db.QueryRow(query, account).Scan(&name)
	err = errors.Wrap(e, "dao: GetUserInfo")
	return
}

func main() {
	connUrl := "root:123456@tcp(localhost:3306)/go_homework?charset=utf8"
	db, err := sql.Open("mysql", connUrl)
	if err != nil {
		fmt.Println("open db failed:", err)
		panic(err)
	}
	fmt.Println("connect db ok")
	dao := &Dao{db: db}
	userHandler := &UserHandler{dao: dao}
	http.HandleFunc("/user/display_name", userHandler.DisplayUserName)
	fmt.Println("listen on 8080")
	http.ListenAndServe(":8080", nil)
}
