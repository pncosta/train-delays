package shared

type StationInfo struct {
	Code        string `json:"code"`
	Designation string `json:"designation"`
}

// stations is a map of station codes to StationInfo
var Stations = map[string]StationInfo{
	// algarve
	"94-73007": {Designation: "Faro", Code: "94-73007"},
	"94-90464": {Designation: "Lagos", Code: "94-90464"},
	"94-73569": {Designation: "Vila Real de Santo António", Code: "94-73569"},
	"94-90290": {Designation: "Portimão", Code: "94-90290"},

	// alentejo
	"94-83006": {Designation: "Évora", Code: "94-83006"},
	"94-75002": {Designation: "Beja", Code: "94-75002"},
	"94-74005": {Designation: "Casa Branca", Code: "94-74005"},
	"94-74278": {Designation: "Vila Nova da Baronia", Code: "94-74278"},

	// lisboa AML
	"94-31039": {Designation: "Lisboa Oriente", Code: "94-31039"},
	"94-30007": {Designation: "Lisboa Santa Apolónia", Code: "94-30007"},
	"94-67025": {Designation: "Alcantara Terra", Code: "94-67025"},
	"94-69039": {Designation: "Alcantara Mar", Code: "94-69039"},
	// Sintra
	"94-59006": {Designation: "Lisboa Rossio", Code: "94-59006"},
	"94-62042": {Designation: "Meleças", Code: "94-62042"},
	"94-61101": {Designation: "Sintra", Code: "94-61101"},
	// Cascais
	"94-69260": {Designation: "Cascais", Code: "94-69260"},
	"94-69005": {Designation: "Cais do Sodré", Code: "94-69005"},
	"94-69179": {Designation: "Oeiras", Code: "94-69179"},
	"94-69229": {Designation: "Sao Pedro Estoril", Code: "94-69229"},
	"94-69120": {Designation: "Caxias", Code: "94-69120"},
	// Azambuja
	"94-33001": {Designation: "Azambuja", Code: "94-33001"},
	"94-31187": {Designation: "Alverca", Code: "94-31187"},
	"94-31310": {Designation: "Castanheira do Ribatejo", Code: "94-31310"},
	// Setubal
	"94-68122": {Designation: "Setúbal", Code: "94-68122"},
	"94-91058": {Designation: "Praias do Sado", Code: "94-91058"},
	"94-95000": {Designation: "Barreiro", Code: "94-95000"},

	// center
	"94-34009": {Designation: "Entroncamento", Code: "94-34009"},
	"94-52001": {Designation: "Abrantes", Code: "94-52001"},
	"94-36004": {Designation: "Coimbra-B", Code: "94-36004"},
	"94-40154": {Designation: "Tomar", Code: "94-40154"},

	// west
	"94-63008": {Designation: "Caldas da Rainha", Code: "94-63008"},
	"94-64113": {Designation: "Figueira da Foz", Code: "94-64113"},
	"94-63560": {Designation: "Leiria", Code: "94-63560"},

	// Beiras
	"94-49460": {Designation: "Vilar Formoso", Code: "94-49460"},
	"94-53009": {Designation: "Castelo Branco", Code: "94-53009"},
	"94-49007": {Designation: "Guarda", Code: "94-49007"},
	"94-54007": {Designation: "Covilhã", Code: "94-54007"},
	"94-52167": {Designation: "Mouriscas", Code: "94-52167"},
	"94-52647": {Designation: "Vila Velha de Rodao", Code: "94-52647"},

	// --- NORTH (Porto Urbanos, Minho, Douro) ---
	"94-2006":  {Designation: "Porto Campanhã", Code: "94-2006"},
	"94-1008":  {Designation: "Porto São Bento", Code: "94-1008"},
	"94-8318":  {Designation: "Penafiel", Code: "94-8318"},
	"94-7005":  {Designation: "Valenca", Code: "94-7005"},
	"94-29157": {Designation: "Braga", Code: "94-29157"},
	"94-24000": {Designation: "Guimarães", Code: "94-24000"},
	"94-38000": {Designation: "Aveiro", Code: "94-38000"},
	"94-18002": {Designation: "Viana do Castelo", Code: "94-18002"},
	"94-6007":  {Designation: "Nine", Code: "94-6007"},
	"94-21071": {Designation: "Leça do Balio", Code: "94-21071"},
	"94-5074": {Designation: "Famalicão", Code: "94-5074"},
	// Aveiro
	"94-38299": {Designation: "Ovar", Code: "94-38299"},
	"94-39040": {Designation: "Granja", Code: "94-39040"},
	// Douro
	"94-12005": {Designation: "Pocinho", Code: "94-12005"},
	"94-10009": {Designation: "Régua", Code: "94-10009"},
	"94-9001":  {Designation: "Marco Canaveses", Code: "94-9001"},
	"94-8383": {Designation: "Caide", Code: "94-8383"},
	"94-11007": {Designation: "Tua", Code: "94-11007"},
	// Vouga
	"94-44016": {Designation: "Espinho Vouga", Code: "94-44016"},
	"94-44339": {Designation: "Oliveira Azemeis", Code: "94-44339"},
	"94-43000": {Designation: "Sernada Vouga", Code: "94-43000"},
	"94-42218": {Designation: "Águeda", Code: "94-42218"},
	"94-42325": {Designation: "Macinhata", Code: "94-42325"},

	// internacional
	"71-22308": {Designation: "Vigo-Guixar", Code: "71-22308"},
	"71-37606": {Designation: "Badajoz", Code: "71-37606"},

}
