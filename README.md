# clienAlarmToast
클리앙 읽지않은 알람이 있을때 Windows 10 노티로 통보해주는 유틸리티

# 실행시 콘솔창이 생기지 않도록 하기 위하여 다음과 같이 빌드 한다.
go build -ldflags="-H windowsgui"
