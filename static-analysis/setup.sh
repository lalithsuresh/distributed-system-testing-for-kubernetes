go get -u golang.org/x/tools/go/packages
go get -u golang.org/x/tools/go/ssa
go get -u github.com/golang-collections/go-datastructures/queue
go get -u gopkg.in/yaml.v2

wget https://dl.k8s.io/v1.18.0/kubernetes-src.tar.gz
mkdir -p $GOPATH/src/k8s.io/kubernetes
tar -xf kubernetes-src.tar.gz -C $GOPATH/src/k8s.io/kubernetes
rm kubernetes-src.tar.gz
