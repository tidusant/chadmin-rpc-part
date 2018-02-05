package main

import (
	"bytes"
	"time"

	"encoding/base64"
	"encoding/json"
	"io/ioutil"

	"github.com/tidusant/c3m-common/c3mcommon"
	"github.com/tidusant/c3m-common/log"
	"github.com/tidusant/c3m-common/lzjs"
	"github.com/tidusant/c3m-common/mystring"
	rpch "github.com/tidusant/chadmin-repo/cuahang"
	"github.com/tidusant/chadmin-repo/models"
	rpsex "github.com/tidusant/chadmin-repo/session"

	"flag"
	"fmt"
	"net"
	"net/http"
	"net/rpc"
	"strconv"
	"strings"
)

const (
	GHTKApiUrl string = "https://services.giaohangtietkiem.vn/"
)

var GHTKCode = map[string]string{
	"-1":  "Hủy Đơn Hàng",
	"1":   "Chưa Tiếp Nhận",
	"2":   "Đã tiếp nhận",
	"3":   "Đã lấy hàng/Đã nhập kho",
	"4":   "Đã điều phối giao hàng/Đang giao hàng",
	"5":   "Đã giao hàng/Chưa đối soát",
	"6":   "Đã đối soát",
	"7":   "Không lấy được hàng",
	"8":   "Hoãn lấy hàng",
	"9":   "Không giao được hàng",
	"10":  "Delay giao hàng",
	"11":  "Đã đối soát công nợ trả hàng",
	"12":  "Đã điều phối lấy hàng/Đang lấy hàng",
	"20":  "Đang trả hàng",
	"21":  "Đã trả hàng",
	"123": "Shipper báo đã lấy hàng",
	"127": "Shipper báo không lấy được hàng",
	"128": "Shipper báo delay lấy hàng",
	"45":  "Shipper báo đã giao hàng",
	"49":  "Shipper báo không giao được giao hàng",
	"410": "Shipper báo delay giao hàng",
}

type GhtkResp struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Order   struct {
		PartnerID            string `json:"partner_id"`
		Label                string `json:"label"`
		Area                 string `json:"area"`
		Fee                  string `json:"fee"`
		InsuranceFee         string `json:"insurance_fee"`
		EstimatedPickTime    string `json:"estimated_pick_time"`
		EstimatedDeliverTime string `json:"estimated_deliver_time"`
	} `json:"order"`
	Fee struct {
		Name         string `json:"name"`
		Fee          int    `json:"fee"`
		InsuranceFee int    `json:"insurance_fee"`
		DeliveryType string `json:"delivery_type"`
	} `json:"fee"`
}

type Arith int

func (t *Arith) Run(data string, result *string) error {
	log.Debugf("Call RPC Partner args:" + data)
	*result = ""
	//parse args
	args := strings.Split(data, "|")

	if len(args) < 3 {
		return nil
	}
	var usex models.UserSession
	usex.Session = args[0]
	usex.Action = args[2]
	info := strings.Split(args[1], "[+]")
	usex.UserID = info[0]
	ShopID := info[1]
	usex.Params = ""
	if len(args) > 3 {
		usex.Params = args[3]
	}

	//	if usex.Action == "c" {
	//		*result = CreateProduct(usex)

	//	} else

	//check shop permission
	shop := rpch.GetShopById(usex.UserID, ShopID)
	if shop.Status == 0 {
		*result = c3mcommon.ReturnJsonMessage("-4", "Shop is disabled.", "", "")
		return nil
	}
	usex.Shop = shop

	if usex.Action == "so" {
		*result = SubmitOrder(usex)
	} else if usex.Action == "po" {
		*result = PrintOrder(usex)
	} else if usex.Action == "co" {
		*result = CancelOrder(usex)
	} else if usex.Action == "vs" {
		*result = ViewShipFee(usex)
	} else if usex.Action == "vl" {
		*result = ViewLog(usex)
	} else {
		*result = c3mcommon.ReturnJsonMessage("-5", "Action not found.", "", "")
	}

	return nil
}

func SubmitOrder(usex models.UserSession) string {

	//default status
	order := rpch.GetOrderByID(usex.Params, usex.Shop.ID.Hex())

	//validate
	if order.ID.Hex() == "" {
		return c3mcommon.ReturnJsonMessage("0", "Order not found!", "", "")
	}
	if len(order.Items) == 0 {
		return c3mcommon.ReturnJsonMessage("0", "Order Empty!", "", "")
	}
	if order.Phone == "" {
		return c3mcommon.ReturnJsonMessage("0", "Phone Empty!", "", "")
	}
	if order.ShipmentCode != "" {
		return c3mcommon.ReturnJsonMessage("0", "Already Submit!", "", "")
	}

	cus := rpch.GetCusByPhone(order.Phone, usex.Shop.ID.Hex())
	if cus.Name == "" {
		return c3mcommon.ReturnJsonMessage("0", "Name Empty!", "", "")
	}
	if cus.City == "" {
		return c3mcommon.ReturnJsonMessage("0", "City Empty!", "", "")
	}
	if cus.District == "" {
		return c3mcommon.ReturnJsonMessage("0", "District Empty!", "", "")
	}
	// if cus.Ward == "" {
	// 	return c3mcommon.ReturnJsonMessage("0", "Ward Empty!", "", "")
	// }
	if order.Address == "" {
		return c3mcommon.ReturnJsonMessage("0", "Address Empty!", "", "")
	}

	//call curl

	// Generated by curl-to-Go: https://mholt.github.io/curl-to-go
	type Product struct {
		Name     string  `json:"name"`
		Weight   float64 `json:"weight"`
		Quantity int     `json:"quantity"`
	}
	type OrderInfo struct {
		ID            string `json:"id"`
		PickName      string `json:"pick_name"`
		PickAddressID string `json:"pick_address_id"`
		PickAddress   string `json:"pick_address"`
		PickWard      string `json:"pick_ward"`
		PickProvince  string `json:"pick_province"`
		PickDistrict  string `json:"pick_district"`
		PickTel       string `json:"pick_tel"`
		Tel           string `json:"tel"`
		Name          string `json:"name"`
		Address       string `json:"address"`
		Province      string `json:"province"`
		District      string `json:"district"`
		Ward          string `json:"ward"`
		IsFreeship    string `json:"is_freeship"`
		PickDate      string `json:"pick_date"`
		PickMoney     int    `json:"pick_money"`
		Note          string `json:"note"`
		Value         int    `json:"value"`
	}

	type Payload struct {
		Products []Product `json:"products"`
		Order    OrderInfo `json:"order"`
	}
	var myProds []Product
	for _, v := range order.Items {
		title, _ := lzjs.DecompressFromBase64(v.Title)

		prod := Product{title, 0.15 * float64(v.Num), v.Num}
		myProds = append(myProds, prod)
	}
	var myOrder OrderInfo
	myOrder.ID = mystring.RandString(8)
	myOrder.PickName = usex.Shop.Name
	myOrder.PickAddressID = usex.Shop.Config.GHTKWareID
	myOrder.PickAddress = usex.Shop.Config.Address
	myOrder.PickProvince = usex.Shop.Config.Province
	myOrder.PickDistrict = usex.Shop.Config.District
	myOrder.PickWard = usex.Shop.Config.Ward
	myOrder.PickTel = usex.Shop.Config.Tel
	myOrder.Name = cus.Name
	myOrder.Tel = cus.Phone
	myOrder.Address = cus.Address
	myOrder.Ward = cus.Ward
	myOrder.District = cus.District
	myOrder.Province = cus.City
	myOrder.IsFreeship = "1"
	myOrder.PickMoney = order.Total
	myOrder.Value = 0
	myOrder.Note = order.Note

	data := Payload{
		Products: myProds,
		Order:    myOrder,
	}

	payloadBytes, err := json.Marshal(data)
	if err != nil {
		// handle err
	}
	body := bytes.NewReader(payloadBytes)

	req, err := http.NewRequest("POST", GHTKApiUrl+"services/shipment/order", body)
	if err != nil {
		// handle err
	}
	//os.Setenv("HTTP_PROXY", "http://127.0.0.1:8888")

	req.Header.Set("Token", usex.Shop.Config.GHTKToken)
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	bodyresp, _ := ioutil.ReadAll(resp.Body)
	bodystr := string(bodyresp)

	var ghtkResp GhtkResp

	//bodystr := `{"success":true,"message":"test","order":{"partner_id":"5a5f04ba50254980160008a0mAH","label":"S264232.MN1.B5.47744801","area":"3","fee":"45000","insurance_fee":"0","estimated_pick_time":"S\u00e1ng 2018-01-28","estimated_deliver_time":"Chi\u1ec1u 2018-01-29","products":[]}}`

	if bodystr == "" {
		return c3mcommon.ReturnJsonMessage("0", "Submit fail!", "", "")
	}

	//var dataresp map[string]json.RawMessage
	err2 := json.Unmarshal([]byte(bodystr), &ghtkResp)
	if !c3mcommon.CheckError(fmt.Sprintf("pasre json %s error ", bodystr), err2) {
		return c3mcommon.ReturnJsonMessage("0", "Submit fail!", "", err.Error())
	}
	if !ghtkResp.Success {
		return c3mcommon.ReturnJsonMessage("0", "Submit fail! "+ghtkResp.Message, "", "")
	}
	order.ShipmentCode = ghtkResp.Order.Label
	order.PartnerShipFee, _ = strconv.Atoi(ghtkResp.Order.Fee)
	order.SearchIndex += order.ShipmentCode
	// order.Name = cus.Name
	// order.Email = cus.Email
	// order.City = cus.City
	// order.District = cus.District
	// order.Ward = cus.Ward
	// order.Address = cus.Address
	// order.CusNote = cus.Note
	rpch.SaveOrder(order)
	info, _ := json.Marshal(order)
	return c3mcommon.ReturnJsonMessage("1", "", "success", string(info))
}

func PrintOrder(usex models.UserSession) string {

	//default status
	order := rpch.GetOrderByID(usex.Params, usex.Shop.ID.Hex())

	//validate
	if order.ID.Hex() == "" {
		return c3mcommon.ReturnJsonMessage("0", "Order not found!", "", "")
	}
	if order.ShipmentCode == "" {
		return c3mcommon.ReturnJsonMessage("0", "Order Not Submit!", "", "")
	}
	//call curl

	body := bytes.NewReader([]byte(""))

	req, err := http.NewRequest("GET", GHTKApiUrl+"services/label/"+order.ShipmentCode, body)
	if err != nil {
		// handle err
	}
	//os.Setenv("HTTP_PROXY", "http://127.0.0.1:8888")

	req.Header.Set("Token", usex.Shop.Config.GHTKToken)
	//req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	bodyresp, _ := ioutil.ReadAll(resp.Body)
	bodystr := string(bodyresp)
	bodyb64 := base64.StdEncoding.EncodeToString(bodyresp)

	var ghtkResp GhtkResp

	//bodystr := `{"success":true,"message":"test","order":{"partner_id":"5a5f04ba50254980160008a0mAH","label":"S264232.MN1.B5.47744801","area":"3","fee":"45000","insurance_fee":"0","estimated_pick_time":"S\u00e1ng 2018-01-28","estimated_deliver_time":"Chi\u1ec1u 2018-01-29","products":[]}}`

	if bodystr == "" {
		return c3mcommon.ReturnJsonMessage("0", "Submit fail!", "", "")
	}

	//var dataresp map[string]json.RawMessage
	err = json.Unmarshal([]byte(bodystr), &ghtkResp)
	if c3mcommon.CheckError(fmt.Sprintf("pasre json %s error ", bodystr), err) {
		if !ghtkResp.Success {
			return c3mcommon.ReturnJsonMessage("0", "Submit fail! "+ghtkResp.Message, "", "")
		}
	}

	return c3mcommon.ReturnJsonMessage("1", "", bodyb64, "")
}

func CancelOrder(usex models.UserSession) string {

	//default status
	order := rpch.GetOrderByID(usex.Params, usex.Shop.ID.Hex())

	//validate
	if order.ID.Hex() == "" {
		return c3mcommon.ReturnJsonMessage("0", "Order not found!", "", "")
	}
	if order.ShipmentCode == "" {
		return c3mcommon.ReturnJsonMessage("0", "Order Not Submit!", "", "")
	}
	//call curl

	body := bytes.NewReader([]byte(""))

	req, err := http.NewRequest("POST", GHTKApiUrl+"services/shipment/cancel/"+order.ShipmentCode, body)
	if err != nil {
		// handle err
	}
	//os.Setenv("HTTP_PROXY", "http://127.0.0.1:8888")

	req.Header.Set("Token", usex.Shop.Config.GHTKToken)
	//req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	bodyresp, _ := ioutil.ReadAll(resp.Body)
	bodystr := string(bodyresp)

	var ghtkResp GhtkResp

	//bodystr := `{"success":true,"message":"test","order":{"partner_id":"5a5f04ba50254980160008a0mAH","label":"S264232.MN1.B5.47744801","area":"3","fee":"45000","insurance_fee":"0","estimated_pick_time":"S\u00e1ng 2018-01-28","estimated_deliver_time":"Chi\u1ec1u 2018-01-29","products":[]}}`

	if bodystr == "" {
		return c3mcommon.ReturnJsonMessage("0", "Cancel partner order fail!", "", "")
	}
	//var dataresp map[string]json.RawMessage
	err = json.Unmarshal([]byte(bodystr), &ghtkResp)
	if c3mcommon.CheckError(fmt.Sprintf("pasre json %s error ", bodystr), err) {
		if !ghtkResp.Success {
			return c3mcommon.ReturnJsonMessage("0", "Cancel partner order fail! "+ghtkResp.Message, "", "")
		}
	}

	return c3mcommon.ReturnJsonMessage("1", "", "cancel partner order success", "")
}

func ViewShipFee(usex models.UserSession) string {
	var order models.Order
	err := json.Unmarshal([]byte(usex.Params), &order)
	if !c3mcommon.CheckError("ord parse json", err) {
		return c3mcommon.ReturnJsonMessage("0", "ord parse fail", "", "")
	}

	if len(order.Items) == 0 {
		return c3mcommon.ReturnJsonMessage("0", "Order Empty!", "", "")
	}

	if order.City == "" {
		return c3mcommon.ReturnJsonMessage("0", "City Empty!", "", "")
	}
	if order.District == "" {
		return c3mcommon.ReturnJsonMessage("0", "District Empty!", "", "")
	}

	//call curl

	totalweight := 0
	for _, v := range order.Items {
		totalweight += 150 * v.Num
	}

	body := bytes.NewReader([]byte(""))

	req, err := http.NewRequest("POST", GHTKApiUrl+"services/shipment/fee", body)
	if err != nil {
		// handle err
	}
	querystr := req.URL.Query()
	querystr.Add("pick_province", usex.Shop.Config.Province)
	querystr.Add("pick_district", usex.Shop.Config.District)
	querystr.Add("pick_ward", usex.Shop.Config.Ward)
	querystr.Add("pick_address", usex.Shop.Config.Address)
	querystr.Add("province", order.City)
	querystr.Add("district", order.District)
	querystr.Add("ward", order.Ward)
	querystr.Add("address", order.Address)
	querystr.Add("weight", strconv.Itoa(totalweight))
	req.URL.RawQuery = querystr.Encode()

	//os.Setenv("HTTP_PROXY", "http://127.0.0.1:8888")

	req.Header.Set("Token", usex.Shop.Config.GHTKToken)
	//req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		// handle err
	}
	defer resp.Body.Close()

	bodyresp, _ := ioutil.ReadAll(resp.Body)
	bodystr := string(bodyresp)

	var ghtkResp GhtkResp

	//bodystr := `{"success":true,"message":"test","order":{"partner_id":"5a5f04ba50254980160008a0mAH","label":"S264232.MN1.B5.47744801","area":"3","fee":"45000","insurance_fee":"0","estimated_pick_time":"S\u00e1ng 2018-01-28","estimated_deliver_time":"Chi\u1ec1u 2018-01-29","products":[]}}`

	if bodystr == "" {
		return c3mcommon.ReturnJsonMessage("0", "Get ship fee fail!", "", "")
	}

	//var dataresp map[string]json.RawMessage
	err = json.Unmarshal([]byte(bodystr), &ghtkResp)
	if c3mcommon.CheckError(fmt.Sprintf("pasre json %s error ", bodystr), err) {
		if !ghtkResp.Success {
			return c3mcommon.ReturnJsonMessage("0", "Get ship fee fail! "+ghtkResp.Message, "", "")
		}
	}

	return c3mcommon.ReturnJsonMessage("1", "", strconv.Itoa(ghtkResp.Fee.Fee), "")
}

func ViewLog(usex models.UserSession) string {

	//call curl
	whs := rpsex.GetWhookByLabel(usex.Params)
	//whs := rpsex.GetWhookByLabel("S70547.SG5.17K.49857769")
	var args struct {
		LabelID    string `json:"label_id"`
		StatusID   string `json:"status_id"`
		PartnerID  string `json:"partner_id"`
		ActionTime string `json:"action_time"`
		ReasonCode string `json:"reason_code"`
		Reason     string `json:"reason"`
		Weight     string `json:"weight"`
		Fee        string `json:"fee"`
	}
	var logarr string
	//var logarr2 []string
	logarr = `[`
	for _, v := range whs {
		log.Debugf(v.Data)
		json.Unmarshal([]byte(v.Data), &args)
		if args.StatusID != "" {
			t, _ := time.Parse(time.RFC3339, args.ActionTime)
			//logarr2 = append(logarr2, "- "+t.Format("2006-01-02 15:04")+": "+GHTKCode[args.StatusID]+" - "+args.Reason)
			strlog := GHTKCode[args.StatusID] + ` - ` + args.Reason
			strlog = base64.StdEncoding.EncodeToString([]byte(strlog))
			logarr += `{"time":"` + t.Format("2006-01-02 15:04") + `","log":"` + strlog + `"},`
		}
	}
	if len(whs) > 0 {
		logarr = logarr[:len(logarr)-1]
	}
	logarr += `]`
	//logstr, _ := json.Marshal(logarr2)
	return c3mcommon.ReturnJsonMessage("1", "", "logstr", logarr)
}

func main() {
	var port int
	var debug bool
	flag.IntVar(&port, "port", 9889, "help message for flagname")
	flag.BoolVar(&debug, "debug", false, "Indicates if debug messages should be printed in log files")
	flag.Parse()

	//logLevel := log.DebugLevel
	if !debug {
		//logLevel = log.InfoLevel
	}

	// log.SetOutputFile(fmt.Sprintf("adminOrder-"+strconv.Itoa(port)), logLevel)
	// defer log.CloseOutputFile()
	// log.RedirectStdOut()

	//init db
	arith := new(Arith)
	rpc.Register(arith)
	log.Infof("running with port:" + strconv.Itoa(port))

	tcpAddr, err := net.ResolveTCPAddr("tcp", ":"+strconv.Itoa(port))
	c3mcommon.CheckError("rpc dail:", err)

	listener, err := net.ListenTCP("tcp", tcpAddr)
	c3mcommon.CheckError("rpc init listen", err)

	for {
		conn, err := listener.Accept()
		if err != nil {
			continue
		}
		go rpc.ServeConn(conn)
	}
}
