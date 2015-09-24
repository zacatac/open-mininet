package netapps

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"github.com/3d0c/ogo"
	"github.com/3d0c/ogo/protocol/ofp10"
	"github.com/julienschmidt/httprouter"
)

type FlowMods []*ofp10.FlowMod

func Switches(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
	b, err := json.MarshalIndent(ogo.Switches(), "", "      ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	fmt.Fprint(w, "%s\n", string(b))
}

func AddFlow(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	decoder := json.NewDecoder(r.Body)

	// var mods FlowMods
	type payload struct {
		FlowMods FlowMods
	}

	p := payload{FlowMods: make([]*ofp10.FlowMod, 0)}

	err := decoder.Decode(&p)
	if err != nil {
		http.Error(w, fmt.Sprintf("can't decode body: %v", err), http.StatusBadRequest)
		return
	}

	dpid, err := net.ParseMAC(ps.ByName("dpid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if sw, ok := ogo.Switch(dpid); ok {
		for _, mod := range p.FlowMods {
			fmt.Println("mod")
			fmt.Println(mod)
			sw.Send(mod)
		}
	}
}

func TestMod(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	f1 := ofp10.NewFlowMod()

	f1.Match.DLSrc, _ = net.ParseMAC("22:11:11:22:22:22")
	f1.Match.DLDst, _ = net.ParseMAC("55:44:55:66:77:88")
	f1.Match.InPort = 5
	// f1.Flags = ofp10.FC_ADD
	// f1.Match.Wildcards = ofp10.FW_ALL ^ ofp10.FW_DL_SRC ^ ofp10.FW_DL_DST

	f1.AddAction(ofp10.NewActionOutput(1))

	dpid, err := net.ParseMAC(ps.ByName("dpid"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
	}

	if sw, ok := ogo.Switch(dpid); ok {
		sw.Send(f1)
	}

}

func NewWebService(apiOn string) {
	router := httprouter.New()
	router.GET("/switches", Switches)
	router.POST("/switches/:dpid/flows", AddFlow)
	router.GET("/test/:dpid", TestMod)

	log.Fatal(http.ListenAndServe(apiOn, router))
}
