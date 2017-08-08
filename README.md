sniproxy
=======

A SNI proxy implements by golang

it can forward the TLS request to deferent backend by deferent SNI name


*install*

    go get github.com/fangdingjun/sniproxy
    cp $GOPATH/src/github.com/fangdingjun/sniproxy/config.sample.yaml config.yaml
    vim config.yaml
    $GOPATH/bin/sniproxy -c config.yaml

