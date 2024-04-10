# trace-merge

combines traces returned from Tempo. grab traces with something like:

```sh
curl 'http://localhost:3100/tempo/api/search?spss=50&limit=100' --data-urlencode 'q={ resource.service.name = "grafana" } >> {resource.service.name != "grafana" && kind = server } | select(nestedSetLeft, nestedSetRight, nestedSetParent, resource.service.name, name)' > out.json
```

or like this for errors:

```sh
curl 'http://localhost:3100/tempo/api/search?spss=50&limit=100' --data-urlencode 'q={ resource.service.name = "grafana" } >> {resource.service.name != "grafana" && status = error } | select(nestedSetLeft, nestedSetRight, nestedSetParent, resource.service.name, name)' > out.json
```

then merge like so:

```sh
go run ./ out.jsonn
```
