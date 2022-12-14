package tool

import "errors"

const version = `go-CFC-v01.02.00`

const HandshakeCheckStepQ1 = "Hello! I want to shake hands."
const HandshakeCheckStepA1 = "Hi! What's your Info?"
const HandshakeCheckStepQ2 = "Here it is"
const HandshakeCheckStepA2 = "OK! Happy handshake"

const PingMsg = "Ping"
const PongMsg = "Pong"

const TaskQ = "TaskQ"
const TaskA = "TaskA"

const SOpenQ = "sOpenQ"
const SOpenA = "sOpenA"

const DelayQ = "DelayQ"
const DelayA = "DelayA"

var errWaitPack = errors.New("wait pack")
