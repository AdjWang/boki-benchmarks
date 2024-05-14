package frontend

import (
	"log"

	"github.com/eniac/Beldi/internal/hotel/main/data"
	"github.com/eniac/Beldi/pkg/cayonlib"
)

func SendRequest(env *cayonlib.Env, userId string, flightId string, hotelId string) string {
	cayonlib.BeginTxn(env)
	input := map[string]string{
		"hotelId": hotelId,
		"userId":  userId,
	}
	res, instanceId := cayonlib.SyncInvoke(env, data.Thotel(), data.RPCInput{
		Function: "ReserveHotel",
		Input:    input,
	})
	if instanceId == "" || res == nil {
		return "Place Order Fails"
	}
	if !res.(bool) {
		cayonlib.AbortTxn(env)
		return "Place Order Fails"
	}
	input = map[string]string{
		"flightId": flightId,
		"userId":   userId,
	}
	res, instanceId = cayonlib.SyncInvoke(env, data.Tflight(), data.RPCInput{
		Function: "ReserveFlight",
		Input:    input,
	})
	if instanceId == "" || res == nil {
		return "Place Order Fails"
	}
	if !res.(bool) {
		cayonlib.AbortTxn(env)
		return "Place Order Fails"
	}
	input = map[string]string{
		"flightId": flightId,
		"hotelId":  hotelId,
		"userId":   userId,
	}
	cayonlib.CommitTxn(env)
	// DEBUG
	log.Println("frontend commit txn ok")
	instanceId = cayonlib.AsyncInvoke(env, data.Torder(), data.RPCInput{
		Function: "PlaceOrder",
		Input:    input,
	})
	if instanceId == "" {
		// DEBUG
		log.Println("frontend return failure")
		return "Place Order Fails"
	}
	// DEBUG
	log.Println("frontend return success")
	return "Place Order Success"
}
