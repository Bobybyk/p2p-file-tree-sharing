package udptypes

import (
	"fmt"
	"protocoles-internet-2023/crypto"
)

func (bytes UDPMessageBytes) BytesToMessage() UDPMessage {

	udpMsg := UDPMessage{}

	udpMsg.Id += uint32(bytes[0])*(1<<24) + (uint32(bytes[1]) * (1 << 16)) + (uint32(bytes[2]) * (1 << 8)) + uint32(bytes[3])
	udpMsg.Type = bytes[4]
	udpMsg.Length = uint16(int(bytes[5])*256 + int(bytes[6]))

	udpMsg.Body = make([]byte, udpMsg.Length)
	for i := 0; i < int(udpMsg.Length); i++ {
		udpMsg.Body[i] = bytes[i+7]
	}

	if len(bytes) > 7+int(udpMsg.Length) { //message is signed
		udpMsg.Signature = bytes[7+int(udpMsg.Length):]
	}

	//udpMsg.Body = bytes[7:]

	return udpMsg
}

func (udpMsg UDPMessage) MessageToBytes() UDPMessageBytes {
	bytes := make([]byte, 7+udpMsg.Length)

	bytes[0] = byte(udpMsg.Id >> 24)
	bytes[1] = byte(udpMsg.Id >> 16)
	bytes[2] = byte(udpMsg.Id >> 8)
	bytes[3] = byte(udpMsg.Id)

	bytes[4] = udpMsg.Type

	bytes[5] = byte(udpMsg.Length >> 8)
	bytes[6] = byte(udpMsg.Length)

	for i := 0; i < int(udpMsg.Length); i++ {
		bytes[7+i] = udpMsg.Body[i]
	}

	if udpMsg.PrivateKey != nil {
		signature, err := crypto.GenerateSignature(bytes, udpMsg.PrivateKey)
		if err != nil {
			fmt.Println("signature of message failed")
		}
		bytes = append(bytes, signature...)
	}

	return bytes
}

func BytesToHelloBody(bytes []byte) HelloBody {
	body := HelloBody{}

	body.Extensions += int32(bytes[0])*(1<<24) + (int32(bytes[1]) * (1 << 16)) + (int32(bytes[2]) * (1 << 8)) + int32(bytes[3])

	body.Name = ""
	for i := 0; i < len(bytes)-4; i++ {
		body.Name += string(bytes[i+4])
	}

	//body.Name = string(bytes[:len(bytes)-1])

	return body
}

func (body HelloBody) HelloBodyToBytes() UDPMessageBytes {
	bytes := make([]byte, len(body.Name)+4)

	bytes[0] = byte(body.Extensions >> 24)
	bytes[1] = byte(body.Extensions >> 16)
	bytes[2] = byte(body.Extensions >> 8)
	bytes[3] = byte(body.Extensions)

	for i := 0; i < len(body.Name); i++ {
		bytes[i+4] = body.Name[i]
	}

	return bytes
}

func BytesToDatumBody(bytes UDPMessageBytes) DatumBody {
	body := DatumBody{}

	body.Hash = [32]byte(bytes[:32])

	/*for i := 0; i < len(bytes)-32; i++ {
		body.Value[i] = bytes[i+32]
	}*/
	body.Value = bytes[32:]

	return body
}

func (body DatumBody) DatumBodyToBytes() UDPMessageBytes {
	bytes := make([]byte, len(body.Value)+32)

	copy(bytes, body.Hash[:])

	for i := 0; i < len(body.Value); i++ {
		bytes[i+32] = body.Value[i]
	}

	return bytes
}
