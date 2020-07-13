package main

import ( 
    "strings"
    "image"
    "time"
    "io"
    "fmt"
    "strconv"
    "encoding/hex"
    "crypto/md5"

    "gopkg.in/toast.v1"
    "github.com/gocolly/colly"
    "golang.org/x/text/encoding/korean"

    "github.com/aarzilli/nucular"
    "github.com/aarzilli/nucular/style"

    "github.com/aarzilli/nucular/label"
    "github.com/aarzilli/nucular/rect"
)

var (
    euckrEnc = korean.EUCKR.NewEncoder()
    userId nucular.TextEditor
    passEd nucular.TextEditor
    intervalEd nucular.TextEditor
    start bool
    failReason string
    alarmList map[string]bool
 )

func main() {
    passEd.PasswordChar = '*'
    passEd.Flags = nucular.EditField

    intervalEd.Filter = nucular.FilterDecimal
    intervalEd.Buffer = []rune(fmt.Sprintf("%d", 15))

    start = false
    alarmList = make(map[string]bool)

    wnd := nucular.NewMasterWindowSize(0, "ClienAlarm", image.Point{200,200} ,updatefn)
    wnd.SetStyle(style.FromTheme(style.DarkTheme, 2.0))

    go func() {
        for {
                if start == true {
                    var ret bool
                    ret, failReason = getNewAlaram() 
                    if ret == false {
                        wnd.PopupOpen("Error", nucular.WindowMovable|nucular.WindowTitle|nucular.WindowDynamic|nucular.WindowNoScrollbar, rect.Rect{50, 20, 110, 100}, true, errorPopup)                        
                        start = false
                    }                            
                    interval, _ := strconv.Atoi(string(intervalEd.Buffer))
                    if interval > 0 {
                        time.Sleep(time.Duration(interval) * time.Second)
                        fmt.Println(interval)                
                    } else {
                        start = false
                    }                    
                } else {
                    time.Sleep(time.Duration(1) * time.Second)
                }     
        }
    }();


    wnd.Main()


    
}

func updatefn(w *nucular.Window) {

    idLabel, _ := euckrEnc.String("ID")
    pwLabel, _ := euckrEnc.String("PW")
    intervalLabel, _ := euckrEnc.String("Interval-sec")
    runningLabel, _ := euckrEnc.String("Running!")

    w.Row(20).Dynamic(1)

    if start == false {
        w.Label(idLabel, "LC")
        userId.Edit(w)

        w.Label(pwLabel, "LC")
        passEd.Edit(w)

        w.Label(intervalLabel, "LC")
        intervalEd.Edit(w)

        if w.ButtonText("Start") {
            start = true
        }  
    } else {
        w.Label(runningLabel, "LC")
        if w.ButtonText("Stop") {
            start = false
        }  

    }
}

func getNewAlaram() (bool, string) {   
    c := colly.NewCollector()
    csrf := ""

    c.OnHTML("input[name]", func(e *colly.HTMLElement) {
        if e.Attr("name") == "_csrf" {
            csrf = e.Attr("value")
        }
    })
    c.Visit("https://www.clien.net/service/")
    if csrf != "" { 
        login_collector := c.Clone()
        err := login_collector.Post("https://www.clien.net/service/login", map[string]string{"_csrf": csrf, "userId": string(userId.Buffer), "userPassword": string(passEd.Buffer)})
        if err != nil {            
            return false, "Network error";
        }

        // start scraping
        login_collector.Visit("https://www.clien.net/service")           

        alarm_collector := login_collector.Clone()
        alarm_collector.OnHTML("div[class]", func(e *colly.HTMLElement) {
            if e.Attr("class") == "list_item unread cursor" {  
                h := md5.New()
                io.WriteString(h, e.Attr("onclick"))
                key := hex.EncodeToString(h.Sum(nil))                
                if _, ok := alarmList[key]; !ok {
                    sTemp := strings.ReplaceAll(e.Attr("onclick"), "app.commentAlarmLink(", "")
                    sTemp  = strings.ReplaceAll(sTemp, ")", "")
                    sTemp  = strings.ReplaceAll(sTemp, "'", "")
                    sTemp  = strings.ReplaceAll(sTemp, " ", "")
                    Split := strings.Split(sTemp, ",")             

                    sTitle, _ := euckrEnc.String("안읽은알람") 
                    sMenu, _  := euckrEnc.String("보러가기") 
                    sEUCKR, _ := euckrEnc.String(e.ChildText("div div a span")) 
                    notification := toast.Notification{
                        Title:   sTitle,
                        Message: sEUCKR,
                        Actions: []toast.Action{
                            {"protocol", sMenu, "https://www.clien.net/service/board/" + Split[0] + "/" + Split[1]},
                           },
                    }

                    notification.Push()
                    alarmList[key] = true
                }        
                
                
            }
        })

        alarm_page_result := 0
        alarm_collector.OnResponse(func(r *colly.Response) {
            if r.StatusCode != 200 {
                alarm_page_result = 1
            } 
            if strings.Contains(string(r.Body), "self.close()") {
                alarm_page_result = 2
            }
        })
        alarm_collector.Visit("https://www.clien.net/service/getAlarmList")
        if alarm_page_result == 1 {
            return false, "Network error"         
        } 

        if alarm_page_result == 2 {
            return false, "Login error"
        }
    } else {
        return false, "Network error" 
    }

    return true, "OK"
}

func errorPopup(w *nucular.Window) {
    w.Row(25).Dynamic(1)
    w.Label(failReason, "LC")
    w.Row(25).Dynamic(2)
    if w.Button(label.T("OK"), false) {
        w.Close()
    }
}
