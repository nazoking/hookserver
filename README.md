# hookserver

receive github webhook, and run os command like cgi

## install

go get github.com/nazoking/hookserver/cmd/hookserver.go



## Usage


```
Usage of hookserver:
  -addr=":8999":   address and port of hook server
  -default="_all": default script name
  -parser:         json file of payload parse configuration
  -scripts:        script root path
  -secret="":      secret of webhook Signature
```

### 1. create script

```bash
mkdir -p /your/home/hookserver/your-username/your-reponame/
cat <<SCRIPT > /your/home/hookserver/your-username/your-reponame/push
#do mirror
cd /your/home/git/your-reponame
git fetch origin
git push mirror
SCRIPT
chmod u+x /your/home/hookserver/your-username/your-reponame/push

hookserver -scripts=/your/home/hookserver
```

and you go to https://github.com/your-username/your-reponame, set webhook send payload to http://your-host:8999/ .
When repository pushed, and github send hook, hookserver run `/your/home/hookserver/your-username/your-reponame/push` .


## Parser json

You can customize script path by parser json. it schema is below.

```json
{
  "Event Name( X-Github-Event )" : {
    "Path": "template string it direct to script path"
    "Values": {
      "Value name": "value path of webhook json payload"
    }
}
```

### Default parser json

```json
{
  "pull_request":{
    "Path": "{{.BASE_OWNER}}/{{.BASE_REPO}}/pull_request/{{.ACTION}}",
    "Values":{
      "HEAD_OWNER"  :"/pull_request/head/repo/owner/login",
      "HEAD_REPO"   :"/pull_request/head/repo/name",
      "HEAD_BRANCH" :"/pull_request/head/ref",
      "HEAD_SHA"    :"/pull_request/head/sha",
      "BASE_OWNER"  :"/pull_request/base/repo/owner/login",
      "BASE_REPO"   :"/pull_request/base/repo/name",
      "BASE_BRANCH" :"/pull_request/head/ref",
      "BASE_SHA"    :"/pull_request/head/sha",
      "ACTION"      :"/action"
    }
  },
  "push":{
    "Path": "{{.OWNER}}/{{.REPO}}/push/{{.BRANCH}}",
    "Values":{
      "OWNER"  :"/repository/owner/name",
      "REPO"   :"/repository/name",
      "BRANCH" :"/ref"
    }
  }
}
```

## Environment variable and payload

Your hook scripts can use Environment Variables like cgi. `QUERY_STRING`, `HTTP_METHOD`, `HTTP_X_GITHUB_EVENT` ...

Your hook scripts can use palyoad value by reading std-in.

`Values` set by parser is export to script. Your hook scripts can use Environment Variables like `ACTION` set by parser.json.

( Path templete can use header variables like cgi. for example, `QUERY_STRING`, `HTTP_METHOD`, `HTTP_X_GITHUB_EVENT` ... )

## default script

hookserver find script by parser's path. if not found path, script search default script.

if you set option `hookscript -scripts=/your/home/hookserver/ -default="_index"` and, path value is `your-username/your-reponame/push` then, script find

  1. `/your/home/hookserver/your-username/your-reponame/push/_index`
  2. `/your/home/hookserver/your-username/your-reponame/push`
  3. `/your/home/hookserver/your-username/your-reponame/_index`
  4. `/your/home/hookserver/your-username/your-reponame`
  5. `/your/home/hookserver/your-username/_index`
  6. `/your/home/hookserver/your-username`
  7. `/your/home/hookserver/_index`
  8. return http not found status


