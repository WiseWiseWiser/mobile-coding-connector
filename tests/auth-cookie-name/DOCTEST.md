# Configurable management cookie namespaces

## DSN (Domain Specific Notion)

- **Pi**: production auth configured with pi-agent-token and its own fixture credential file.
- **Mac**: separate production auth process using the unchanged ai-critic-token default.
- **Origin hop**: production authproxy with DisableAuth=true; it filters Pi credentials but does not add a mapping login.
- **Browser**: HTTP cookie-jar client with distinct pi.test and mac.test domains mapped only to fixture loopback listeners.
- **Invalid startup**: in-process run.Run rejects invalid names before setting auth configuration or starting services.

## Version

0.0.1

## Decision Tree

```text
management cookie namespace
├── browser-login/                          L3: two real auth processes
│   └── pi-custom-mac-default/               login, refresh, filtering, legacy rejection
└── invalid-startup/                         L2: in-process Run, no mutable global setup
    ├── server-empty/
    ├── server-invalid/
    ├── server-reserved/
    ├── server-secure-prefix/
    ├── keep-alive-empty/
    └── systemctl-empty/
```

The greatest user-visible distinction is first: independent Pi/Mac browser sessions
must survive the origin hop. That scenario requires separate processes because the
production auth configuration is per-process. The helper uses production auth and
proxy handlers but deliberately does not start background services/public tunnels.
It is not a Chrome/UI or live-Pi deployment test. Short rejection paths are L2,
parallel-safe, and do not swap process stdio, environment, or working directory.
Pure name tables, daemon requests, help, systemd generation, and streaming regressions
also live in adjacent Go tests. Legacy help currently calls os.Exit and is isolated
in the Go CLI helper process instead of swapping global stdio.

## How to Run

```sh
doctest vet ./tests/auth-cookie-name
doctest test --label-all ./tests/auth-cookie-name/...
```

```go
import (
    "bufio"
    "bytes"
    "context"
    "encoding/json"
    "fmt"
    "io"
    "net"
    "net/http"
    "net/http/cookiejar"
    "net/url"
    "os"
    "os/exec"
    "path/filepath"
    "testing"
    "time"

    "github.com/xhd2015/ai-critic/run"
    "github.com/xhd2015/doctest/session"
)

type Request struct {
    Mode string
    Args []string
}

type Response struct {
    Error string
    Checks map[string]int
}

func Run(t *testing.T, d *session.Doctest, req *Request) (*Response, error) {
    if req.Mode == "reject" {
        err := run.Run(req.Args)
        resp := &Response{}
        if err != nil { resp.Error = err.Error() }
        return resp, nil
    }
    return runNamespaces(t, d), nil
}

func fixtureProcess(t *testing.T, binary, name, token, backend string) map[string]string {
    t.Helper()
    home := t.TempDir()
    file := filepath.Join(home,"credentials")
    if err := os.WriteFile(file,[]byte(token+"\n"),0600); err != nil { t.Fatal(err) }
    args := []string{"--auth-cookie-name",name,"--credentials-file",file}
    if backend!="" { args=append(args,"--backend",backend) }
    ctx,cancel := context.WithTimeout(context.Background(),45*time.Second)
    cmd := exec.CommandContext(ctx,binary,args...)
    cmd.Env = append(os.Environ(),"HOME="+home)
    cmd.Dir = home
    var stderr bytes.Buffer
    cmd.Stderr=&stderr
    stdout,err := cmd.StdoutPipe()
    if err!=nil {cancel();t.Fatal(err)}
    if err=cmd.Start();err!=nil {cancel();t.Fatal(err)}
    t.Cleanup(func(){
        _=cmd.Process.Signal(os.Interrupt)
        _=cmd.Wait()
        cancel()
        if stderr.Len()>0 {t.Logf("fixture diagnostic: %s",stderr.String())}
    })
    scanner:=bufio.NewScanner(stdout)
    if !scanner.Scan(){t.Fatal("fixture exited before publishing its bound listeners")}
    var addresses map[string]string
    if err=json.Unmarshal(scanner.Bytes(),&addresses);err!=nil {t.Fatal(err)}
    return addresses
}

func runNamespaces(t *testing.T, d *session.Doctest) *Response {
    t.Helper()
    moduleRoot:=filepath.Clean(filepath.Join(d.DOCTEST_ROOT,"..",".."))
    binary:=filepath.Join(t.TempDir(),"cookie-name-fixture")
    buildCtx,cancel:=context.WithTimeout(context.Background(),90*time.Second)
    defer cancel()
    build:=exec.CommandContext(buildCtx,"go","build","-o",binary,"./tests/auth-cookie-name/testdata/fixture")
    build.Dir=moduleRoot
    if output,err:=build.CombinedOutput();err!=nil {t.Fatalf("build isolated fixture: %v\n%s",err,output)}
    mac:=fixtureProcess(t,binary,"ai-critic-token","fixture-mac-token","")
    pi:=fixtureProcess(t,binary,"pi-agent-token","fixture-pi-token",mac["management"])
    // Distinct browser domains, even though the fixture sockets are loopback.
    // Do not rely on ports to isolate cookies: browsers do not do that.
    routes:=map[string]string{}
    publicURL:=func(raw,host string)string{
        parsed,err:=url.Parse(raw);if err!=nil {t.Fatal(err)}
        actual:=parsed.Host
        parsed.Host=net.JoinHostPort(host,parsed.Port())
        routes[parsed.Host]=actual
        return parsed.String()
    }
    piURL:=publicURL(pi["management"],"pi.test")
    macURL:=publicURL(pi["proxy"],"mac.test")
    sharedPiURL:=publicURL(pi["management"],"shared.test")
    sharedMacURL:=publicURL(pi["proxy"],"shared.test")
    tr:=&http.Transport{DialContext:func(ctx context.Context,network,address string)(net.Conn,error){
        actual,ok:=routes[address];if !ok{return nil,fmt.Errorf("unexpected fixture destination")}
        return (&net.Dialer{}).DialContext(ctx,network,actual)
    }}
    defer tr.CloseIdleConnections()
    jar,_:=cookiejar.New(nil)
    browser:=&http.Client{Transport:tr,Jar:jar,Timeout:5*time.Second}
    bare:=&http.Client{Transport:tr,Timeout:5*time.Second}
    resp:=&Response{Checks:map[string]int{}}
    call:=func(client *http.Client,base,path,bearer,cookieName,cookieValue string) int {
        t.Helper()
        req,_:=http.NewRequest("GET",base+path,nil)
        if bearer!=""{req.Header.Set("Authorization","Bearer "+bearer)}
        if cookieName!=""{req.AddCookie(&http.Cookie{Name:cookieName,Value:cookieValue})}
        res,err:=client.Do(req);if err!=nil {t.Fatal(err)}
        io.Copy(io.Discard,res.Body);res.Body.Close()
        return res.StatusCode
    }
    login:=func(base,token,name string){
        t.Helper()
        data,_:=json.Marshal(map[string]string{"username":"fixture","password":token})
        res,err:=browser.Post(base+"/api/login","application/json",bytes.NewReader(data));if err!=nil {t.Fatal(err)}
        io.Copy(io.Discard,res.Body);res.Body.Close()
        if res.StatusCode!=200 {t.Fatalf("login failed: %d",res.StatusCode)}
        found:=false
        for _,cookie:=range res.Cookies(){
            if cookie.Name==name&&cookie.MaxAge>0&&cookie.HttpOnly&&cookie.Path=="/" {found=true}
        }
        if !found {t.Fatal("login did not issue the configured HttpOnly cookie")}
    }
    resp.Checks["pi-anonymous"]=call(bare,piURL,"/api/private","","","")
    resp.Checks["mac-anonymous"]=call(bare,macURL,"/api/private","","","")
    login(piURL,"fixture-pi-token","pi-agent-token")
    login(macURL,"fixture-mac-token","ai-critic-token")
    resp.Checks["pi-cookie"]=call(browser,piURL,"/api/private","","","")
    resp.Checks["mac-cookie"]=call(browser,macURL,"/api/private","","","")
    resp.Checks["mac-refresh"]=call(browser,macURL,"/api/auth/check","","","")
    resp.Checks["pi-legacy-cookie"]=call(bare,piURL,"/api/private","","ai-critic-token","fixture-pi-token")
    resp.Checks["pi-bearer"]=call(bare,piURL,"/api/private","fixture-pi-token","","")
    resp.Checks["mac-bearer"]=call(bare,macURL,"/api/private","fixture-mac-token","","")
    resp.Checks["pi-bearer-not-forwarded"]=call(bare,macURL,"/api/private","fixture-pi-token","","")
    req,_:=http.NewRequest("GET",macURL+"/_fixture/headers",nil)
    req.AddCookie(&http.Cookie{Name:"pi-agent-token",Value:"fixture-pi-token"})
    observed,err:=browser.Do(req);if err!=nil{t.Fatal(err)}
    var headers struct{Cookies []string;Bearer bool}
    err=json.NewDecoder(observed.Body).Decode(&headers);observed.Body.Close();if err!=nil{t.Fatal(err)}
    haveMac:=false
    for _,name:=range headers.Cookies{
        if name=="pi-agent-token"{t.Fatal("Pi cookie leaked to Mac")}
        if name=="ai-critic-token"{haveMac=true}
    }
    if !haveMac {t.Fatal("Mac cookie was stripped")}
    // Relogin never expires an unrelated namespace, even for same-host instances
    // on different ports: browser cookies are host/path-scoped, not port-scoped.
    login(piURL,"fixture-pi-token","pi-agent-token")
    resp.Checks["mac-after-pi-relogin"]=call(browser,macURL,"/api/private","","","")
    login(sharedMacURL,"fixture-mac-token","ai-critic-token")
    login(sharedPiURL,"fixture-pi-token","pi-agent-token")
    resp.Checks["same-host-mac-after-pi-login"]=call(browser,sharedMacURL,"/api/private","","","")
    resp.Checks["same-host-pi-cookie"]=call(browser,sharedPiURL,"/api/private","","","")
    return resp
}
```
