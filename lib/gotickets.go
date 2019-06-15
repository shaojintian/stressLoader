package lib

import (
	"errors"
	"fmt"
)

//GoTickets  : goroutine  tickets pool
type GoTickets interface {
	//take
	Take()
	//return
	Return()
	//Active ?
	Active() bool
	//tickets numbers
	Total() uint32
	//remain tickets
	Remainder() uint32
}

// implement  GoTickets

type myGoTickets struct{

	total 		uint32   			//tickets numbers
	ticketCh 	chan struct{}		//ticket  pool
	active 		bool

}


//implement  funcs for  myGoTickets
func (gt *myGoTickets) init (total  uint32) bool{
	// already  active ---->  not init again
	if gt.active{
		return false
	}
	if total == 0 {
		return false
	}
	ch := make(chan struct{} , total)
	n := int(total)
	for i:=0; i<n; i++ {
		ch <- struct{}{}
	}

	gt.total =total
	gt.ticketCh = ch
	gt.active = true

	return gt.active


}


func (gt *myGoTickets) Take(){
	<-gt.ticketCh
}
func  (gt *myGoTickets) Return(){
	gt.ticketCh <- struct{}{}
}
func (gt *myGoTickets) Active() bool {
	return gt.active
}

func (gt *myGoTickets) Total() uint32 {
	return gt.total
}

func (gt *myGoTickets) Remainder() uint32 {
	return uint32(len(gt.ticketCh))
}


//NewGoTickets :  create  a new  tickets pool

func NewGoTickets(total uint32) (GoTickets,error){
	gt := myGoTickets{}
	if !gt.init(total){
		err:=
			fmt.Sprintln("init failure (total = %d)\n",total)
		return nil , errors.New(err)
	}

	return   &gt ,nil
}
















