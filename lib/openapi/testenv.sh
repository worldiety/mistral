cat << EOF > build/testmod/main.go
package main

func main(){
  var t BucketGroupType
  _ = t
}

EOF