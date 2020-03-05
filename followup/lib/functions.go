package lib

import "os"

func Env_defined(key string) bool {
        _, exists := os.LookupEnv(key)
        return exists
}
