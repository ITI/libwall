package libwall

import (
    "fmt"
    "errors"
    serial "github.com/tarm/goserial"

)


// Packet Format
// +--------+---------+----+-----------+------+----------+
// | Header | Command | ID | len(data) | data | checksum |
// +--------+---------+----+-----------+------+----------+
// |  0xAA  |         |    |           |      |  Below   |
// +--------+---------+----+-----------+------+----------+
//
// Checksum
// (sum(data, id, len(data), data)) % 256

const (
    ON byte = 0x01
    OFF byte = 0x00
    ALL byte = 0xfe
)

var ControlCodes = map[string]byte {
    "power":        0x11,
    "volume":       0x12,
    "source":       0x14,
    "mode":         0x18,
    "size":         0x19,
    "pip":          0x3c,
    "autoAdjust":   0x3d,
    "vwallMode":    0x5c,
    "safety":       0x5d,
    "wall":         0x89,
}

// DVI_VIDEO, HDMI1_PC, HDMI2_PC → Get Only
// In the case of MagicInfo, only possible with models include MagicInfo
// In the case of TV, only possible with models include TV.
var Sources = map[string]byte {
    "pc":           0x14,
    "bnc":          0x1e,
    "dvi":          0x18,
    "av":           0x0c,
    "svideo":       0x04,
    "component":    0x08,
    "magicinfo":    0x20,
//  "dvi_video":    0x1f,
//  "rf_tv":        0x30,
    "hdmi1":        0x21,
//  "hdmi1_pc":     0x22,
    "hdmi2":        0x23,
//  "hdmi2_pc":     0x24,
    "displayport":  0x25,
}

type Panel struct {
    ID, Position, X, Y byte
    Port *serial.Config
    Debug bool
}
func NewPanel(id byte, port string, debug bool) (*Panel) {
    p := new(Panel)
    p.ID = id
    p.Port = new(serial.Config)
    p.Port.Name = port
    p.Port.Baud = 9600
    p.Debug = debug
    return p
}
func (p *Panel) Set(command string, value ...byte) (error) {
    cmd, ok := ControlCodes[command]
    if !ok {
        return errors.New(fmt.Sprintf("%v is not an available command",
            command))
    }
    pkt := p.mkpkt(cmd, value)
    if p.Debug {
        fmt.Printf("%v <<id: %v, cmd:%v(%v) %v>>\n", pkt, p.ID, command, cmd, value)
    } else {
        s, err := serial.OpenPort(p.Port)
        if err != nil {
            return err
        }
        _, err = s.Write(pkt)
        if err != nil {
            return err
        }
        s.Close()
    }

    return nil
}
func (p *Panel) csum(cmd byte, data []byte) (byte) {
    x := int32(p.ID + cmd + byte(len(data)))
    for _, d := range data {
        x += int32(d)
    }
    r := x % 256
    return byte(r)

}
func (p *Panel) mkpkt(cmd byte, data []byte) ([]byte) {
    r := []byte{0xaa, cmd, p.ID, byte(len(data))}
    for _, d := range data {
        r = append(r, byte(d))
    }
    r = append(r, p.csum(cmd, data))
    return r
}


type Wall struct {
    Panels []*Panel
}
func (wall Wall) On() (error) {
    // The logic here is odd:
    // iter the panels, setting wall on
    // if any panel pos is not set, turn all wall paneles to wall(off)
    //
    // This is a hack for not knowing state
    for _,p := range wall.Panels {
        if p.X == 0 || p.Y==0 {
            wall.Off()
            return errors.New(fmt.Sprintf("Panel %v does not have a position",
                p.ID))
        }
        err := p.Set("wall", wallCode(p.X, p.Y), p.Position)
        if err != nil {
            return err
        }
    }

    return nil
}
func (wall Wall) Off() (error){
    for _, p := range wall.Panels {
        err := p.Set("wall", OFF)
        if err != nil {
            return err
        }
    }
    return nil
}

func wallCode(x, y byte) (code byte) {
    return (0x01 * y) + (0x10 *x)
}
