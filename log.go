package nacos

import "fmt"

type LogInterface interface {
	Error(string, ...interface{})
	Warn(string, ...interface{})
	Info(string, ...interface{})
	Debug(string, ...interface{})
	SetLevel(string)
}

var _ LogInterface = &logger{}

type logger struct {
	level int8
}

func newDefaultLogger(level string) LogInterface {
	l := &logger{}
	l.SetLevel(level)
	return l
}

func (c *logger) SetLevel(level string) {
	var lv int8 = 2
	switch level {
	case "error":
		lv = 4
	case "warn":
		lv = 3
	case "info":
		lv = 2
	case "debug":
		lv = 1
	default:
	}
	c.level = lv
}

func lvToString(lv int8) string {
	switch lv {
	case 4:
		return "[error]"
	case 3:
		return "[warn]"
	case 2:
		return "[info]"
	case 1:
		return "[debug]"
	}
	return "[info]"
}

func (c *logger) println(lv int8, msg string, params ...interface{}) {
	if c.level <= lv {
		v := append([]interface{}{}, lvToString(lv), msg)
		if len(params) > 0 {
			v = append(v, params...)
		}
		fmt.Println(v...)
	}
}

func (c *logger) Error(msg string, params ...interface{}) {
	c.println(4, msg, params...)
}

func (c *logger) Warn(msg string, params ...interface{}) {
	c.println(3, msg, params...)
}

func (c *logger) Info(msg string, params ...interface{}) {
	c.println(2, msg, params...)
}

func (c *logger) Debug(msg string, params ...interface{}) {
	c.println(1, msg, params...)
}
