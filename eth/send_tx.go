package eth

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"reflect"
	"strings"

	"github.com/AlexNa-Holdings/web3pro/bus"
	"github.com/AlexNa-Holdings/web3pro/cmn"
	"github.com/AlexNa-Holdings/web3pro/gocui"
	"github.com/ava-labs/coreth/accounts/abi"
	"github.com/ethereum/go-ethereum"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/common/hexutil"
	"github.com/ethereum/go-ethereum/core/types"
	"github.com/rs/zerolog/log"
)

func sendTx(msg *bus.Message) (string, error) {
	req, ok := msg.Data.(*bus.B_EthSendTx)
	if !ok {
		return "", fmt.Errorf("invalid tx: %v", msg.Data)
	}

	w := cmn.CurrentWallet
	if w == nil {
		return "", errors.New("no wallet")
	}

	b := w.GetBlockchain(req.ChainId)
	if b == nil {
		return "", fmt.Errorf("send_tx: blockchain not found: %v", req.ChainId)
	}

	from := w.GetAddress(req.From.String())
	if from == nil {
		return "", fmt.Errorf("address from not found: %v", req.From)
	}

	template, err := BuildHailToSendTxTemplate(b, from, req.To, req.Amount, req.Data, nil)
	if err != nil {
		log.Error().Err(err).Msg("Error building send-tx hail template")
		bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", err))
		return "", err
	}

	nt, _ := w.GetNativeToken(b)

	tx, err := BuildTx(b, w.GetSigner(from.Signer), from, req.To, req.Amount, req.Data)
	if err != nil {
		log.Error().Err(err).Msg("Error building transaction")
		bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", err))
		return "", err
	}

	confirmed := false

	msg.Fetch("ui", "hail", &bus.B_Hail{
		Title:    "Send Tx",
		Template: template,
		OnOk: func(m *bus.Message) bool {
			confirmed = true
			return true
		},
		OnOverHotspot: func(m *bus.Message, v *gocui.View, hs *gocui.Hotspot) {
			cmn.StandardOnOverHotspot(v, hs)
		},
		OnClickHotspot: func(m *bus.Message, v *gocui.View, hs *gocui.Hotspot) {
			rebuild := false
			if hs != nil {
				switch hs.Value {
				case "button edit_gas_price":
					go editFee(m, v, tx, nt, func(newGasPrice *big.Int) {
						tx.GasPrice().Set(newGasPrice)
						template, err := BuildHailToSendTxTemplate(b, from, req.To, req.Amount, req.Data, newGasPrice)
						if err != nil {
							log.Error().Err(err).Msg("Error building hail template")
							bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", err))
							return
						}

						v.GetGui().UpdateAsync(func(*gocui.Gui) error {
							hail, ok := m.Data.(*bus.B_Hail)
							if ok {
								hail.Template = template
								v.RenderTemplate(template)
							}
							return nil
						})
					})
				case "button edit_contract":
					go editContract(m, v, req.To, func() {
						template, err := BuildHailToSendTxTemplate(b, from, req.To, req.Amount, req.Data, nil)
						if err != nil {
							log.Error().Err(err).Msg("Error building hail template")
							bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", err))
							return
						}
						v.GetGui().UpdateAsync(func(*gocui.Gui) error {
							hail, ok := m.Data.(*bus.B_Hail)
							if ok {
								hail.Template = template
								v.RenderTemplate(template)
							}
							return nil
						})
					})
				case "button download_contract":
					resp := msg.Fetch("explorer", "download-contract", &bus.B_ExplorerDownloadContract{
						Blockchain: b.Name,
						Address:    req.To,
					})
					if resp.Error != nil {
						bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", resp.Error))
					} else {
						rebuild = true
						bus.Send("ui", "notify", "Contract downloaded")
					}
				case "button trust":
					err := w.TrustContract(req.To)
					if err != nil {
						bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", err))
					} else {
						rebuild = true
						bus.Send("ui", "notify", "Contract trusted")
					}
				case "button untrust":
					err := w.UntrustContract(req.To)
					if err != nil {
						bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", err))
					} else {
						rebuild = true
						bus.Send("ui", "notify", "Contract untrusted")
					}

				default:
					cmn.StandardOnClickHotspot(v, hs)
				}

				if rebuild {
					template, err := BuildHailToSendTxTemplate(b, from, req.To, req.Amount, req.Data, nil)
					if err != nil {
						log.Error().Err(err).Msg("Error building hail template")
						bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", err))
						return
					}
					v.GetGui().UpdateAsync(func(*gocui.Gui) error {
						hail, ok := m.Data.(*bus.B_Hail)
						if ok {
							hail.Template = template
							v.RenderTemplate(template)
						}
						return nil
					})
				}
			}
		},
	})

	if !confirmed {
		return "", fmt.Errorf("rejected by user")
	}

	signer := w.GetSigner(from.Signer)
	if signer == nil {
		return "", fmt.Errorf("signer not found: %v", from.Signer)
	}

	sign_res := msg.Fetch("signer", "sign-tx", &bus.B_SignerSignTx{
		Type:      signer.Type,
		Name:      signer.Name,
		MasterKey: signer.MasterKey,
		Chain:     b.Name,
		Tx:        tx,
		From:      from.Address,
		Path:      from.Path,
	})

	if sign_res.Error != nil {
		return "", fmt.Errorf("error signing transaction: %v", sign_res.Error)
	}

	signedTx, ok := sign_res.Data.(*types.Transaction)
	if !ok {
		log.Error().Msgf("sendTx: Cannot convert to transaction. Data:(%v)", sign_res.Data)
		return "", errors.New("cannot convert to transaction")
	}

	hash, err := SendSignedTx(signedTx)
	if err != nil {
		log.Error().Err(err).Msg("sendTx: Cannot send tx")
		bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", err))
		return "", err
	}

	bus.Send("ui", "notify", "Transaction sent: "+hash)

	return hash, nil
}

func BuildTx(b *cmn.Blockchain, s *cmn.Signer, from *cmn.Address, to common.Address,
	amount *big.Int, data []byte) (*types.Transaction, error) {

	if from.Signer != s.Name {
		log.Error().Msgf("BuildTxTransfer: Signer mismatch. Token:(%s) Blockchain:(%s)", from.Signer, s.Name)
		return nil, errors.New("signer mismatch")
	}

	client, err := getEthClient(b)
	if err != nil {
		log.Error().Msgf("BuildTxTransfer: Failed to open client: %v", err)
		return nil, err
	}

	nonce, err := client.PendingNonceAt(context.Background(), from.Address)
	if err != nil {
		log.Error().Msgf("BuildTxTransfer: Cannot get nonce. Error:(%v)", err)
		return nil, err
	}

	msg := ethereum.CallMsg{
		From:  from.Address,
		To:    &to,
		Gas:   0, // Set to 0 for gas estimation
		Value: amount,
		Data:  data,
	}

	gasLimit, err := client.EstimateGas(context.Background(), msg)
	if err != nil {
		log.Error().Msgf("BuildTxTransfer: Cannot estimate gas. Error:(%v)", err)
		return nil, err
	}

	// // Simulate the transaction
	// _, err = client.CallContract(context.Background(), msg, nil)
	// if err != nil {
	// 	log.Error().Msgf("BuildTxTransfer: Cannot simulate transaction. Error:(%v)", err)
	// 	return nil, err
	// }

	priorityFee, err := client.SuggestGasTipCap(context.Background())
	if err != nil {
		log.Error().Err(err).Msg("Failed to suggest gas tip cap")
		return nil, err
	}

	// Get the latest block to determine the base fee
	block, err := client.BlockByNumber(context.Background(), nil) // Get the latest block
	if err != nil {
		log.Error().Err(err).Msg("Failed to get the latest block")
		return nil, err
	}

	// Base fee is included in the block header (introduced in EIP-1559)
	baseFee := block.BaseFee()
	// Calculate the MaxFeePerGas based on base fee and priority fee
	// For example, you might want to set MaxFeePerGas to be slightly higher than baseFee + priorityFee
	maxFeePerGas := new(big.Int).Add(baseFee, priorityFee)
	buffer := big.NewInt(2) // Set a buffer (optional) to ensure transaction gets processed
	maxFeePerGas = maxFeePerGas.Mul(maxFeePerGas, buffer)

	tx := types.NewTx(&types.DynamicFeeTx{
		ChainID:   big.NewInt(int64(b.ChainId)),
		Nonce:     nonce,
		To:        &to,
		Value:     amount,
		Gas:       gasLimit,
		GasFeeCap: maxFeePerGas,
		GasTipCap: priorityFee,
		Data:      data,
	})

	return tx, nil

}

func BuildHailToSendTxTemplate(b *cmn.Blockchain, from *cmn.Address, to common.Address,
	amount *big.Int, data []byte, suggested_gas_price *big.Int) (string, error) {
	if cmn.CurrentWallet == nil {
		return "", errors.New("no wallet")
	}

	w := cmn.CurrentWallet

	nt, err := w.GetNativeToken(b)
	if err != nil {
		return "", err
	}

	s := w.GetSigner(from.Signer)
	if s == nil {
		return "", errors.New("signer not found")
	}

	contract_name := "(unknown)"
	contract := w.GetContract(to)
	if contract != nil {
		contract_name = contract.Name
	}

	contract_trusted := w.IsContractTrusted(to)

	dollars := ""
	if nt.Price > 0 {
		dollars = cmn.TagShortDollarLink(nt.Price * nt.Float64(amount))
	} else {
		dollars = "(unknown)"
	}

	tx, err := BuildTx(b, s, from, to, amount, data)

	if err != nil {
		log.Error().Err(err).Msg("Error building transaction")
		return "", err
	}

	gas_price := tx.GasFeeCap()
	gp_change := ""
	if suggested_gas_price != nil && suggested_gas_price.Cmp(gas_price) != 0 {
		if suggested_gas_price.Cmp(gas_price) < 0 {
			percents := new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(gas_price, suggested_gas_price), big.NewInt(100)), gas_price)
			f := float64(percents.Int64())
			gp_change = fmt.Sprintf(` <color fg:green>↓%2.2f%%`, f)
			if f > 10 {
				gp_change += "\n<c><blink>TOO LOW</blink></c>"
			}
			gp_change += `</color>`
		} else {
			percents := new(big.Int).Div(new(big.Int).Mul(new(big.Int).Sub(suggested_gas_price, gas_price), big.NewInt(100)), gas_price)
			f := float64(percents.Int64())
			gp_change = fmt.Sprintf(` <color fg:red>↑%2.2f%%`, f)
			if f > 10 {
				gp_change += "\n<c><blink>TOO HIGH</blink></c>"
			}
			gp_change += `</color>`
		}

		log.Debug().Msgf("go_change: %v", gp_change)

		gas_price = suggested_gas_price
	}
	total_gas := new(big.Int).Mul(new(big.Int).SetUint64(tx.Gas()), gas_price)

	total_fee_s := "(unknown)"
	if nt.Price > 0 {
		total_fee_dollars := nt.Price * cmn.Float64(total_gas, 18)
		// total_fee_s = cmn.FormatDollarsNormal(total_fee_dollars)
		total_fee_s = cmn.TagShortDollarLink(total_fee_dollars)
	}

	color_tag := "<blink><color fg:red>"
	color_tag_end := "</color></blink>"
	if contract_trusted {
		color_tag = "<color fg:green>"
		color_tag_end = "</color>"
	}

	toolbar := `<l text:'` + gocui.ICON_EDIT + `' action:'button edit_contract' tip:"Edit Contract">`
	toolbar += `<l text:'` + gocui.ICON_DOWNLOAD + `' action:'button download_contract' tip:"Download Contract Code">`

	if cmn.IsContractDownloaded(to) {
		action := "system \"" + cmn.Config.Editor + "\" \"" + cmn.DataFolder + "/contracts/" + to.String() + "\""
		toolbar += `<l text:'` + gocui.ICON_VSC + `' action:'` + action + `' tip:"Open Contract Code">`
	}

	if w.IsContractTrusted(to) {
		toolbar += `<l text:'` + gocui.ICON_NO_ENTRY + `' action:'button untrust' tip:"Mark as not trusted">`
	} else {
		toolbar += `<l text:'` + gocui.ICON_TRUST + `' action:'button trust' tip:"Mark as trusted">`
	}

	burl, _ := strings.CutSuffix(b.ExplorerUrl, "/")

	toolbar += `<l text:'` + gocui.ICON_LINK + `' action:'open ` + burl + "/address/" + to.String() + `' tip:"Open in Explorer">`

	call_details := buildCallDetails(w, tx, to)

	return `  Blockchain: ` + b.Name + `
        From: ` + cmn.TagAddressShortLink(from.Address) + " " + from.Name + `
      Amount: ` + nt.Value2Str(amount) + " " + nt.Symbol + `
   Amount($): ` + dollars + ` 
      Signer: ` + s.Name + " (" + s.Type + ")" + `
<line text:Contract>
     Address: ` + cmn.TagAddressShortLink(to) + `
        Name: ` + color_tag + contract_name + color_tag_end + `
<c>` + toolbar + `</c>` + call_details + `
<line text:Fee> 
   Gas Limit: ` + cmn.TagUint64Link(tx.Gas()) + ` 
   Gas Price: ` + cmn.TagValueSymbolLink(gas_price, nt) + " " +
		` <l text:` + gocui.ICON_EDIT + ` action:'button edit_gas_price' tip:"Edit Fee">` + gp_change + `
   Total Fee: ` + cmn.TagValueSymbolLink(total_gas, nt) + `
Total Fee($): ` + total_fee_s + `
<c>
` +
		`<button text:Send id:ok bgcolor:g.HelpBgColor color:g.HelpFgColor tip:"send tokens">  ` +
		`<button text:Reject id:cancel bgcolor:g.ErrorFgColor tip:"reject transaction">`, nil
}

func editContract(m *bus.Message, v *gocui.View, address common.Address, on_close func()) {
	w := cmn.CurrentWallet
	if w == nil {
		bus.Send("ui", "notify-error", "No wallet")
		return
	}

	contract := w.GetContract(address)
	if contract == nil {
		contract = &cmn.Contract{
			Name:    "",
			Trusted: false,
		}
	}

	m.Fetch("ui", "popup", &gocui.Popup{
		Title: "Edit Contract",
		Template: `<c>
 Name: <input id:name size:16 value:"` + contract.Name + `"> 
<c>
<button text:'Ok' id:ok> <button text:'Cancel' id:cancel>
`,
		OnClickHotspot: func(v *gocui.View, hs *gocui.Hotspot) {
			if hs != nil {
				switch hs.Value {
				case "button ok":
					contract.Name = v.GetInput("name")
					w.Contracts[address] = contract
					err := w.Save()
					if err != nil {
						bus.Send("ui", "notify-error", fmt.Sprintf("Error: %v", err))
					} else {
						v.GetGui().HidePopup()
					}
				case "button cancel":
					v.GetGui().HidePopup()
				}
			}
		},
		OnClose: func(*gocui.View) {
			if on_close != nil {
				on_close()
			}
		},
	})
}

func buildCallDetails(w *cmn.Wallet, tx *types.Transaction, to common.Address) string {
	r := ""

	if !cmn.IsContractDownloaded(to) {
		return r + "(no ABI)"
	}

	// Load and parse ABI
	path := cmn.DataFolder + "/abi/" + to.String() + ".json"
	abiJSON, err := os.ReadFile(path)
	if err != nil {
		return r + "(error reading ABI)"
	}

	parsedABI, err := abi.JSON(bytes.NewReader(abiJSON))
	if err != nil {
		return r + "(error parsing ABI)"
	}

	data := tx.Data()
	if len(data) < 4 {
		return r + "(insufficient data for function selector)"
	}

	// Extract the first 4 bytes for the function selector
	functionSelector := data[:4]

	// Match the function selector to a function in the ABI
	var function abi.Method
	found := false
	for _, method := range parsedABI.Methods {
		if bytes.Equal(functionSelector, method.ID) {
			function = method
			found = true
			break
		}
	}

	if !found {
		return r + "(function not found)"
	}

	r += functionDetails(parsedABI, function, data)

	return r
}

func functionDetails(parsedABI abi.ABI, function abi.Method, data []byte) string {

	r := `
      Method: `

	r += fmt.Sprintf("<l text:'%v' tip:'Copy function name' action:'copy %v'>", function.Name, function.Name) + `
  Parameters: ` + cmn.TagLink(gocui.ICON_COPY, "copy "+hexutil.Encode(data[4:]), "Copy data") + ` 
`
	// Decode the parameters of the function
	args := data[4:] // The rest of the data contains the function parameters
	params, err := function.Inputs.Unpack(args)
	if err != nil {
		return r + "(error unpacking)"
	}

	// Display the parameters
	for i, param := range params {
		r += fmt.Sprintf("%12s: ", function.Inputs[i].Name)
		switch function.Inputs[i].Type.T {
		case abi.IntTy, abi.UintTy:
			r += fmt.Sprintf("%v\n", param)
		case abi.BoolTy:
			r += fmt.Sprintf("%t\n", param)
		case abi.StringTy:
			r += fmt.Sprintf("%q\n", param)
		case abi.AddressTy:
			address := param.(common.Address)
			r += cmn.TagAddressShortLink(address) + "\n"
		case abi.FixedBytesTy, abi.BytesTy:
			bytesValue := param.([]byte)
			r += fmt.Sprintf("0x%x\n", bytesValue)
		case abi.SliceTy, abi.ArrayTy:
			// Handling array/slice of values
			v := reflect.ValueOf(param)
			if v.Len() > 0 &&
				(v.Index(0).Kind() == reflect.Uint8 ||
					v.Index(0).Kind() == reflect.Int8) {
				// If it's a byte array, display as hex string
				bytesValue := param.([]byte)
				r += formatBytesAsHex(bytesValue)
			} else {
				r += "["
				for j := 0; j < v.Len(); j++ {
					if j > 0 {
						r += ", "
					}
					elem := v.Index(j)
					if elem.Kind() == reflect.Slice && elem.Type().Elem().Kind() == reflect.Uint8 {
						var f abi.Method
						found := false
						if function.Name == "multicall" && len(elem.Interface().([]byte)) > 4 {
							// Match the function selector to a function in the ABI
							functionSelector := elem.Interface().([]byte)[:4]
							for _, method := range parsedABI.Methods {
								if bytes.Equal(functionSelector, method.ID) {
									f = method
									found = true
									break
								}
							}
						}

						if found {
							r += functionDetails(parsedABI, f, elem.Interface().([]byte))

						} else {
							r += formatBytesAsHex(elem.Interface().([]byte))
						}
					} else {
						r += fmt.Sprintf("%v", elem.Interface())
					}
				}
				r += "]\n"
			}
		case abi.TupleTy:
			// Handling tuples, including structs
			r += "(\n "
			v := reflect.ValueOf(param)
			for j := 0; j < v.NumField(); j++ {
				if j > 0 {
					r += ",\n "
				}
				field := v.Field(j)
				r += fmt.Sprintf("%v", field.Interface())
			}
			r += ")\n"
		case abi.HashTy:
			hash := param.([32]byte)
			r += fmt.Sprintf("0x%x\n", hash)
		case abi.FixedPointTy:
			r += fmt.Sprintf("%v (FixedPoint)\n", param)
		case abi.FunctionTy:
			functionData := param.([24]byte)
			r += fmt.Sprintf("0x%x\n", functionData)
		default:
			r += "(unknown type)\n"
		}
	}

	return r
}

func formatBytesAsHex(bytesValue []byte) string {
	MAX_IN_LINE := 16

	result := "\n"

	for i := 0; i < len(bytesValue); i += MAX_IN_LINE {
		end := i + MAX_IN_LINE
		if end > len(bytesValue) {
			end = len(bytesValue)
		}

		line := "0x"
		if i > 0 {
			line = ""
			result += "\n  "
		}
		result += line + fmt.Sprintf("%x", bytesValue[i:end])
	}

	return result
}
