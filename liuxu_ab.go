package main

import (
    "net/http"
    "net/url"
    "time"
    "fmt"
    "flag"
    "sort"
    "os"
    "text/template"
    "log"
)

const tmpl = `
#########################
发送请求数：{{.Count}}
并发数：{{.Concurrency}}
#########################
{{range $idx, $val := .Report_times}}
{{tmpl_idx $idx}}%的请求在{{$val}}毫秒内完成
{{end}}`

var concurrency = flag.Int("c", 1, "并发数")
var count = flag.Int("n", 1, "总请求数")

func main() {
    flag.Parse()

    // 命令行参数验证
    if len(flag.Args()) != 1 {
        fmt.Println("使用方法：liuxu_ab [-flags] url")
        return
    }
    url := flag.Args()[0]
    if !is_url(url) {
        fmt.Println("错误的url，示例：https://www.baidu.com/")
        return
    }

    var req_times []int64

    // 并发请求
    con_ch := make(chan int64, *concurrency)
    defer close(con_ch)
    repeat := *count / *concurrency
    remainder := *count % *concurrency
    for i:=0; i<repeat; i++ {
        for i:=0; i<*concurrency; i++ {
            go req(&url, con_ch)
        }
        for i:=0; i<*concurrency; i++ {
            req_times = append(req_times, <- con_ch)
        }
    }
    for i:=0; i<remainder; i++ {
        go req(&url, con_ch)
    }
    for i:=0; i<remainder; i++ {
        req_times = append(req_times, <- con_ch)
    }

    // 排序
    sort.Slice(req_times, func (i, j int) bool { return req_times[i] < req_times[j] })
    report_times := get_report_nums(req_times)
    var data struct {
        Count int
        Concurrency int
        Report_times []int64
    }
    data.Count = *count
    data.Concurrency = *concurrency
    data.Report_times = report_times

    //输出
    var report = template.Must(template.New("report").Funcs(template.FuncMap{"tmpl_idx": tmpl_idx}).Parse(tmpl))
    if err := report.Execute(os.Stdout, data); err != nil {
        log.Fatal(err)
    }
    return
}

// 发送请求
func req(url *string, ch chan<- int64) {
    start := time.Now()
    resp, err := http.Get(*url)
    defer resp.Body.Close()
    if err != nil {
        fmt.Println(err)
        ch <- 0
        return
    }
    ch <- time.Since(start).Milliseconds()
    return
}

// 判断url格式
func is_url(str string) bool {
    u, err := url.Parse(str)
    return err == nil && u.Scheme != "" && u.Host != ""
}

// 获取输出数据
func get_report_nums(req_times []int64) []int64 {
    count := len(req_times)
    var report_times []int64
    if count <= 10 {
        for i, _ := range req_times {
            report_times = append(report_times, req_times[i])
        }
    } else {
        num := count / 10
        for i:=0; i<10; i++ {
            report_times = append(report_times, req_times[(i+1)*num])
        }
    }
    return report_times
}

// template计算百分比
func tmpl_idx(idx int) int {
    return (idx + 1) * 10
}
