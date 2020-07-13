package main

import ( 
    "strings"
    "image"
    "time"
    "io"
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
    start bool
    failReason string
    alarmList map[string]bool
 )

func main() {
    passEd.PasswordChar = '*'
    passEd.Flags = nucular.EditField

    start = false
    alarmList = make(map[string]bool)

    wnd := nucular.NewMasterWindowSize(0, "ClienAlarm", image.Point{200,138} ,updatefn)
    wnd.SetStyle(style.FromTheme(style.DarkTheme, 2.0))

    go func() {
        for {
                if start == true {
                    getNewAlaram() 
                    if start == false {
                        wnd.PopupOpen("Error", nucular.WindowMovable|nucular.WindowTitle|nucular.WindowDynamic|nucular.WindowNoScrollbar, rect.Rect{50, 20, 110, 100}, true, errorPopup)                        
                    }                            
                }                
                time.Sleep(10 * time.Second)
        }
    }();


    wnd.Main()


    
}

func updatefn(w *nucular.Window) {

    idLabel, _ := euckrEnc.String("ID")
    pwLabel, _ := euckrEnc.String("PW")

    w.Row(20).Dynamic(1)

    w.Label(idLabel, "LC")
    userId.Edit(w)

    w.Label(pwLabel, "LC")
    passEd.Edit(w) 

    if start == false {
        if w.ButtonText("Start") {
            start = true
        }  
    } else {
        if w.ButtonText("Stop") {
            start = false
        }  

    }
}

func getNewAlaram() {   
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
            start = false;
            failReason = "Network error"
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

        alarm_collector.OnResponse(func(r *colly.Response) {
            if r.StatusCode != 200 {
                start = false
                failReason = "Network error"
            } 
            if strings.Contains(string(r.Body), "self.close()") {
                start = false
                failReason = "Login error"
            }
        })
        alarm_collector.Visit("https://www.clien.net/service/getAlarmList")
    } else {
        start = false;
        failReason = "Network error"
    }
}

func errorPopup(w *nucular.Window) {
    w.Row(25).Dynamic(1)
    w.Label(failReason, "LC")
    w.Row(25).Dynamic(2)
    if w.Button(label.T("OK"), false) {
        w.Close()
    }
}
