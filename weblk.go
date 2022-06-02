package main

import (
	"bytes"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
)

type user struct {
	login string
	pass  string
	key   string
}

var users = [2]user{
	{"admin", "123", generateKey("admin", "123")},
	{"guest", "123", generateKey("guest", "123")},
}

var authPage = `<doctype html><html lang="ru"><meta http-equiv="Content-Type" content="text/html; charset=UTF-8"><head>
	<title>LK</title></head><body><form method="POST" action='/'><input type="text" name="applogin"/>
	<input type="password" name="apppass"/><input type="submit"/></form></body></html>`

var wrongPage = `<doctype html><html lang="ru"><meta http-equiv="Content-Type" content="text/html; charset=UTF-8"><head>
<title>LK</title></head><body>Самфинг вронх! Либо логин, либо пароль.</body></html>`
var wrongPage2 = `<doctype html><html lang="ru"><meta http-equiv="Content-Type" content="text/html; charset=UTF-8"><head>
<title>LK</title></head><body>Сессия истекла. Необходимо авторизоваться заново.</body>
<script>
function rd() {
	location.href = '/';
}
setTimeout(rd, 3000);
</script></html>`

var mainPage = `<doctype html><html lang="ru"><meta http-equiv="Content-Type" content="text/html; charset=UTF-8"><head>
<title>LK</title></head><body>Привет, $utulaya, все норм, мы тебя посчитали</body></html>`

func root(w http.ResponseWriter, r *http.Request) {

	var displayThis string

	displayThis = authPage

	tkeyCookie, err := r.Cookie("tkey")
	if err != nil {
		fmt.Println("Не удалось получить cookie")
		if r.PostFormValue("applogin") != "" {
			userCandidate, present := findUser(generateKey(r.PostFormValue("applogin"), r.PostFormValue("apppass")))
			if present > 0 {
				cookie := &http.Cookie{
					Name:   "tkey",
					Value:  generateKey(r.PostFormValue("applogin"), r.PostFormValue("apppass")),
					MaxAge: 300,
				}
				http.SetCookie(w, cookie)
				displayThis = strings.Replace(mainPage, "$utulaya", userCandidate.login, -1)
			} else {
				displayThis = wrongPage
			}

		}

	} else {
		fmt.Println("\nЗначение tkey:", tkeyCookie)
		userCandidate, present := findUser(tkeyCookie.Value)
		if present > 0 {
			displayThis = strings.Replace(mainPage, "$utulaya", userCandidate.login, -1)
		}

	}

	w.Header().Set("Content-Type", "text/html")
	io.WriteString(w, displayThis)
}

func uploadFile(w http.ResponseWriter, r *http.Request) {
	mr, err := r.MultipartReader()
	if err != nil {
		fmt.Sprintln(err)
		fmt.Fprintln(w, err)
		return
	}
	values := make(map[string][]string, 0)
	maxValueBytes := int64(10 << 20)
	for {
		part, err := mr.NextPart()
		if err == io.EOF {
			break
		}
		name := part.FormName()
		if name == "" {
			continue
		}
		fileName := part.FileName()
		var b bytes.Buffer
		if fileName == "" {
			n, err := io.CopyN(&b, part, maxValueBytes)
			if err != nil && err != io.EOF {
				fmt.Sprintln(err)
				fmt.Fprintln(w, err)
				return
			}
			maxValueBytes -= n
			if maxValueBytes <= 0 {
				msg := "переполненьице"
				fmt.Fprint(w, msg)
				return
			}
			values[name] = append(values[name], b.String())
		}
		dst, err := os.Create("/upload/" + fileName)
		defer dst.Close()

		for {
			buffer := make([]byte, 100000)
			cBytes, err := part.Read(buffer)
			if err == io.EOF {
				break
			}
			dst.Write(buffer[0:cBytes])
		}
	}
}

func findUser(k string) (user, int) {
	//здесь обращаемся к таблице с пользователями
	var ret user
	count := 0
	for _, value := range users {
		if value.key == k {
			ret = value
			count = 1
		}
	}
	return ret, count
}

func generateKey(l, p string) string {
	hsh_s := sha256.Sum256([]byte(l))
	hsh_p := sha256.Sum256([]byte(p))
	x := fmt.Sprintf("%x", hsh_s)
	y := fmt.Sprintf("%x", hsh_p)
	tkey := x + "." + y
	fmt.Println("Сгенерирован ключ для пары", l, p, ":", tkey)
	return tkey
}

func main() {
	http.HandleFunc("/", root)
	http.HandleFunc("/up", uploadFile)

	http.ListenAndServe(":80", nil)
}
