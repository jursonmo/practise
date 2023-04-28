package proto

import (
	"bytes"
	"encoding/binary"
	"log"

	"errors"
	"fmt"
	"io"
	"math"

	"github.com/jursonmo/practise/pkg/encoding"
	_ "github.com/jursonmo/practise/pkg/encoding/json"
)

type ProtoPkg struct {
	ProtoHeader
	options []ProtoHeaderOption
	Payload []byte
}

type ProtoHeader struct {
	Ver  byte // ver:bit [0-3], cmd: bit [4-7]
	Type byte //PkgType [0-3], PayLoadType,[4-7]
	Hlen uint16
	Plen uint32 //payload len
}

type ProtoHeaderOption struct {
	T byte
	L uint16 // l < 127, if >127, need 2byte
	V []byte
}

type PayloadType byte

var BufSizeErr = errors.New("buf size error")

const (
	ProtoHeaderSize = 8

	//version
	Ver1 = 1

	//PkgType: packag Message type
	Msg        = 0
	Ping       = 1
	Pong       = 2
	Auth       = 3
	MaxPkgType = 15

	//options type
	AuthReq  = 1
	AuthOk   = 2
	AuthFail = 3

	//payload Type, 0 mean raw Binary
	RawBinary      = 0
	JSON           = 1
	MaxPayloadType = 15
)

var (
	ErrPkgType     = fmt.Errorf("ErrPkgType, max:%d", MaxPkgType)
	ErrPayloadType = fmt.Errorf("ErrPayloadType, max:%d", MaxPayloadType)
)

var PayloadNameTypeMap = map[string]byte{
	"json": JSON,
}

var PayloadTypeNameMap = map[byte]string{
	JSON: "json",
}

func GetPayloadTypeByName(name string) byte {
	v, _ := PayloadNameTypeMap[name]
	return v
}

func GetPayloadTypeName(b byte) string {
	v, _ := PayloadTypeNameMap[b]
	return v
}

func NewProtoPkg() *ProtoPkg {
	//todo: sync.Pool
	return &ProtoPkg{}
}

func NewPingPkg(payload []byte, opts ...ProtoHeaderOption) (*ProtoPkg, error) {
	return EncodePkg(payload, Ping, 0, opts...)
}

func NewPongPkg(payload []byte, opts ...ProtoHeaderOption) (*ProtoPkg, error) {
	return EncodePkg(payload, Pong, 0, opts...)
}

func NewAuthReqPkg(d []byte) (*ProtoPkg, error) {
	v := d
	authOpt := ProtoHeaderOption{T: AuthReq, L: uint16(len(v)), V: v}
	return EncodePkg(nil, Auth, 0, []ProtoHeaderOption{authOpt}...)
}

func NewAuthRespPkg(resp []byte, authOk bool) (*ProtoPkg, error) {
	t := AuthOk
	if !authOk {
		t = AuthFail
	}
	v := resp
	authOpt := ProtoHeaderOption{T: byte(t), L: uint16(len(v)), V: v}
	return EncodePkg(nil, Auth, 0, []ProtoHeaderOption{authOpt}...)
}

func EncodePkg(payload []byte, pkgType byte, payloadType PayloadType, opts ...ProtoHeaderOption) (*ProtoPkg, error) {
	if pkgType > MaxPkgType {
		return nil, ErrPkgType
	}

	if payloadType > MaxPayloadType {
		return nil, ErrPayloadType
	}

	phType := pkgType | byte(payloadType<<4)
	optsLen := uint16(OptionsLen(opts))
	p := &ProtoPkg{ProtoHeader: ProtoHeader{Ver: Ver1, Type: byte(phType), Hlen: ProtoHeaderSize + optsLen, Plen: uint32(len(payload))}}
	p.options = opts
	p.Payload = payload
	return p, nil
}

func (p *ProtoPkg) String() string {
	return fmt.Sprintf("%s,%s", p.ProtoHeader.String(), p.OptionsInfo())
}

func (p *ProtoPkg) OptionsInfo() string {
	if len(p.options) == 0 {
		return ""
	}
	ss := "option:"
	for _, p := range p.options {
		ss = fmt.Sprintf("%s,%s", ss, p.String())
	}
	return ss
}

func (p *ProtoPkg) Bytes() []byte {
	var err error
	ph := &p.ProtoHeader
	buf := make([]byte, ph.Hlen, uint32(ph.Hlen)+ph.Plen)
	_, err = ph.EncodeWithBuf(buf[:ProtoHeaderSize])
	if err != nil {
		return nil
	}
	//optsData := EncodeOpts(p.options)
	//buf = append(buf, optsData...)
	b := EncodeOptsWithBuffer(p.options, buf[ProtoHeaderSize:ProtoHeaderSize:ph.Hlen])
	//check
	if len(b) != int(ph.Hlen)-ProtoHeaderSize {
		log.Panicf("------")
	}
	buf = append(buf, p.Payload...)
	//check
	if len(buf) != int(ph.Hlen)+int(ph.Plen) {
		log.Panicf("len(buf):%d,  int(ph.Hlen)+int(ph.Plen)=%d+%d=%d", len(buf), ph.Hlen, ph.Plen, uint32(ph.Hlen)+ph.Plen)
	}
	return buf
}

func (p *ProtoPkg) Marshal(v interface{}, codecName string) error {
	codec := encoding.GetCodec(codecName)
	if codec == nil {
		return fmt.Errorf("GetCodec err, payload type name:%s", codecName)
	}

	t := GetPayloadTypeByName(codecName)
	if t == 0 {
		return fmt.Errorf("GetPayloadTypeByName err, codec name:%s", codecName)
	}
	//fmt.Printf("Marshal payload t:%d\n", t)

	payload, err := codec.Marshal(v)
	if err != nil {
		return err
	}
	p.ProtoHeader = ProtoHeader{
		Ver: Ver1,
		//Type: Msg,
		Hlen: ProtoHeaderSize,
		Plen: uint32(len(payload)),
	}
	p.SetPkgType(Msg)
	p.SetPayloadType(t)
	p.Payload = payload
	return nil
}

func (p *ProtoPkg) Unmarshal(v interface{}) error {
	t := p.PayloadType()
	tn := GetPayloadTypeName(byte(t))
	if tn == "" {
		return fmt.Errorf("GetPayloadTypeName err, payload type:%v", t)
	}
	codec := encoding.GetCodec(tn)
	if codec == nil {
		return fmt.Errorf("GetCodec err, payload type name:%s", tn)
	}
	return codec.Unmarshal(p.Payload, v)
}

func (p *ProtoPkg) Decode(r io.Reader) error {
	phBuf := make([]byte, ProtoHeaderSize)
	n, err := io.ReadFull(r, phBuf)
	if err != nil {
		return err
	}
	if n != ProtoHeaderSize {
		err = fmt.Errorf("n:%d, ProtoHeaderSize:%d", n, ProtoHeaderSize)
		return err
	}
	ph := ProtoHeaderBinary(phBuf).Decode()
	//fmt.Printf("Decoding ph:%v\n", &ph)

	ver := ph.GetVer()
	if ver == Ver1 {
		p.ProtoHeader = ph
		//have options?
		if ph.Hlen > ProtoHeaderSize {
			buf := make([]byte, int(ph.Hlen-ProtoHeaderSize))
			_, err = io.ReadFull(r, buf)
			if err != nil {
				return err
			}
			//log.Printf("options buf, len:%d, buf[0]:%d\n", len(buf), buf[0])
			opts := make([]ProtoHeaderOption, 0, 1)
			buffer := bytes.NewBuffer(buf)
			i := 0
			for {
				i++
				opt := ProtoHeaderOption{}
				err := opt.DecodeFromBuffer(buffer)
				if err == io.EOF { //read the end of buffer, it is normal, just break and go on
					break
				}
				if err != nil {
					log.Printf("DecodeFromBuffer:%v", err)
					return err
				}
				//log.Printf("Decode, opt_index:%d, opt:%v\n", i, &opt)
				opts = append(opts, opt)
			}
			p.options = opts
		}
		//have payload ?
		if ph.Plen > 0 {
			buf := make([]byte, int(ph.Plen))
			_, err = io.ReadFull(r, buf)
			if err != nil {
				return err
			}
			p.Payload = buf
		}
		return nil
	}
	return fmt.Errorf("unspport pkg version:%d", ver)
}

func (opt *ProtoHeaderOption) String() string {
	return fmt.Sprintf("type:%s,len:%d", opt.TypeName(), opt.L)
}

func (opt *ProtoHeaderOption) TypeName() string {
	switch opt.T {
	case AuthReq:
		return "AuthRequest"
	case AuthOk:
		return "AuthOk"
	case AuthFail:
		return "AuthFial"
	default:
		return fmt.Sprintf("Unkown Option Type:%d", opt.T)
	}
}

func (po ProtoHeaderOption) Len() int {
	l := len(po.V)
	if len(po.V) < 127 {
		return l + 2 // len(V)+ T(1byte)+ L(1byte)
	}
	return l + 3 // len(V)+ T(1byte)+ L(2byte)
}

//var
func (po ProtoHeaderOption) WriteToBuffer(b *bytes.Buffer) {
	//check
	if len(po.V) != int(po.L) {
		log.Panicf("len(po.V) != int(po.L), opt:%s", po.String())
	}
	b.WriteByte(po.T)
	if po.L < 127 {
		b.WriteByte(byte(po.L))
	} else {
		l1 := byte(po.L) | 1<<7 //add high bite, 添加高位
		b.WriteByte(l1)
		l2 := byte(po.L >> 8)
		b.WriteByte(l2)
	}
	b.Write(po.V)
}

func (po *ProtoHeaderOption) DecodeFromBuffer(b *bytes.Buffer) (err error) {
	po.T, err = b.ReadByte()
	if err != nil {
		return
	}
	l, err := b.ReadByte()
	if err != nil {
		return
	}
	if l < 127 {
		po.L = uint16(l)
	} else {
		l2, err := b.ReadByte()
		if err != nil {
			return err
		}
		l1 := uint16((l << 1) >> 1) //clear high bit
		po.L = (uint16(l2) << 8) | l1
	}
	po.V = b.Next(int(po.L))
	//check
	if len(po.V) != int(po.L) {
		log.Panicf("opt.L:%d, len(opt.V):%d, not the same", int(po.L), len(po.V))
	}
	return nil
}

func OptionsLen(pos []ProtoHeaderOption) int {
	sum := 0
	for _, po := range pos {
		sum += po.Len()
	}
	if sum > math.MaxUint16 {
		//todo:
		log.Panicf("options sum len(%d) > math.MaxUint16(%d) ", sum, math.MaxUint16)
	}
	return sum
}

func EncodeOpts(pos []ProtoHeaderOption) []byte {
	if len(pos) == 0 {
		return nil
	}
	l := OptionsLen(pos)
	//buf := make([]byte, l)
	buf := make([]byte, 0, l)
	buffer := bytes.NewBuffer(buf)
	for _, po := range pos {
		po.WriteToBuffer(buffer)
	}
	b := buffer.Bytes()

	//check
	if len(b) != len(buf) {
		log.Printf("notice, ProtoHeaderOption len:%d, but after encode len:%d\n", l, len(b))
		panic("")
	}
	return b
}

func EncodeOptsWithBuffer(pos []ProtoHeaderOption, buf []byte) []byte {
	if len(pos) == 0 {
		return nil
	}
	l := OptionsLen(pos)
	//check
	if len(buf) != 0 || cap(buf) != l {
		log.Panicf("notice, ProtoHeaderOption len:%d, but buf len:%d, cap:%d", l, len(buf), cap(buf))
	}

	buffer := bytes.NewBuffer(buf)
	for _, po := range pos {
		po.WriteToBuffer(buffer)
	}
	return buffer.Bytes()
}

func DecodeOpts(b []byte) (pos []ProtoHeaderOption) {
	//todo:

	return
}

type ProtoHeaderBinary []byte

func (pbb ProtoHeaderBinary) Decode() (ph ProtoHeader) {
	// buf := bytes.NewBuffer(pbb)
	// ph.Ver, _ = buf.ReadByte()
	// ph.Type, _ = buf.ReadByte()
	// binary.Read(buf, binary.BigEndian, &ph.Hlen)
	// binary.Read(buf, binary.BigEndian, &ph.Plen)

	ph.Ver = pbb[0]
	ph.Type = pbb[1]
	ph.Hlen = binary.BigEndian.Uint16(pbb[2:])
	ph.Plen = binary.BigEndian.Uint32(pbb[4:])
	return
}

func (ph *ProtoHeader) String() string {
	return fmt.Sprintf("%s, ver:%d, Type:%d(pkgType:%d, payloadType:%d), Hlen:%d, PayloadLen:%d",
		ph.PkgTypeName(), ph.GetVer(), ph.Type, ph.PkgType(), ph.PayloadType(), ph.Hlen, ph.Plen)
}

func (ph *ProtoHeader) GetVer() byte {
	return (ph.Ver << 4) >> 4
}

func (ph *ProtoHeader) PkgTypeName() string {
	switch ph.PkgType() {
	case Msg:
		return "user msg"
	case Ping:
		return "ping"
	case Pong:
		return "pong"
	case Auth:
		return "auth"
	default:
		return "unknown"
	}
}

func (ph *ProtoHeader) PkgType() byte {
	return (ph.Type << 4) >> 4
}

func (ph *ProtoHeader) SetPkgType(t byte) {
	ph.Type = ph.Type&0xf0 | t
}

func (ph *ProtoHeader) PayloadType() PayloadType {
	return PayloadType(ph.Type >> 4)
}

func (ph *ProtoHeader) SetPayloadType(t byte) {
	ph.Type = ph.Type&0x0f | t<<4
}

func (ph *ProtoHeader) Encode() (ProtoHeaderBinary, error) {
	return ph.EncodeWithBuf(make([]byte, ProtoHeaderSize)) //todo: use sync.Pool
}

func (ph *ProtoHeader) EncodeWithBuf(b []byte) (ProtoHeaderBinary, error) {
	if b == nil || len(b) < ProtoHeaderSize {
		return nil, BufSizeErr
	}

	b[0] = ph.Ver
	b[1] = ph.Type
	binary.BigEndian.PutUint16(b[2:], ph.Hlen)
	binary.BigEndian.PutUint32(b[4:], ph.Plen)

	return ProtoHeaderBinary(b), nil
}
