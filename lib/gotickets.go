package lib

import "fmt"

type GoTickets interface {
	Take()             //拿走一张票
	Return()           //归还一张票
	Active() bool      //票池是否启用
	Total() uint32     //票池总票数
	Remainder() uint32 //票池剩余票数
}

type myGoTickets struct {
	total    uint32
	isActive bool
	ticketCh chan struct{}
}

func NewGoTickets(total uint32) (GoTickets, error) {
	gt := &myGoTickets{}
	if !gt.init(total) {
		err := fmt.Errorf("Go tickets pool can not be initialized total %d", total)
		return nil, err
	}
	return gt, nil
}

func (gt *myGoTickets) init(total uint32) bool {
	if gt.isActive {
		return false
	}
	if total == 0 {
		return false
	}
	gt.ticketCh = make(chan struct{}, total)
	for i := 0; i < int(total); i++ {
		gt.ticketCh <- struct{}{}
	}
	gt.isActive = true
	gt.total = total
	return true
}

func (gt *myGoTickets) Take() {

}

func (gt *myGoTickets) Return() {

}

func (gt *myGoTickets) Active() bool {
	return gt.isActive
}

func (gt *myGoTickets) Total() uint32 {
	return gt.total
}

func (gt *myGoTickets) Remainder() uint32 {

}
