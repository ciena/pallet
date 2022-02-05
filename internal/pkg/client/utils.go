package client

import (
	"fmt"
	"hash/fnv"
	"k8s.io/apimachinery/pkg/util/rand"
)

func getName(namespace, podset string) string {
	hash := fnv.New32a()

	hash.Write([]byte(namespace))
	hash.Write([]byte(podset))

	name := podset + "-" + rand.SafeEncodeString(fmt.Sprint(hash.Sum32()))

	return name
}
