package test

import (
	"myTips/tipstocks/app/utils"
	"testing"
)

// TestLoadConf : passed!
func TestLoadConf(t *testing.T) {
	// dir, _ := os.Getwd()
	// println(dir)
	_ = utils.LoadConf("../utils/config.ini") // from app/test
}
