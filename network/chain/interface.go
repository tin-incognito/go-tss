package chain

type Chain interface {
	Start() error
	ProcessTxsIn()
}
