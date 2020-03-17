package musicplayer

import (
	"github.com/faiface/beep"
	"github.com/faiface/beep/effects"
	"github.com/faiface/beep/mp3"
	"github.com/faiface/beep/speaker"
	"log"
	"os"
	"time"
)

type MusicEntry struct {
	Id          string                // 编号
	Name        string                // 歌名
	Artist      string                // 作者
	Source      string                // 位置
	Type        string                // 类型
	Filestream  *os.File              // 文件流
	Format      beep.Format           // 文件信息
	Streamer    beep.StreamSeekCloser // 流信息
	done        chan bool             // 结束信号
	ctrl        *beep.Ctrl            // 控制器
	volume      *effects.Volume       // 带音量的控制
	paused      chan bool             // 暂停标志
	increaseVol chan bool             // 增加音量
	decreaseVol chan bool             // 减少音量
	progress    float64               // 进度值
	over        bool                  // 播放结束
	active      bool                  // 是否激活
}

func (me *MusicEntry) Open() {
	me.over = false
	var err error
	me.Filestream, err = os.Open(me.Source)
	if err != nil {
		log.Fatal(err)
	}
	me.Streamer, me.Format, err = mp3.Decode(me.Filestream)
	if err != nil {
		log.Fatal(err)
	}
	speaker.Init(me.Format.SampleRate, me.Format.SampleRate.N(time.Second/10))
	me.done = make(chan bool)
	me.paused = make(chan bool)
	me.increaseVol = make(chan bool)
	me.decreaseVol = make(chan bool)
	me.ctrl = &beep.Ctrl{Streamer: beep.Seq(me.Streamer, beep.Callback(func() {
		me.done <- true
	})), Paused: false}

	me.volume = &effects.Volume{
		Streamer: me.ctrl,
		Base:     2,
		Volume:   0,
		Silent:   false,
	}
	me.active = true
}

func (me *MusicEntry) Play() {
	defer me.Streamer.Close()
	speaker.Play(me.volume)
	for {
		select {
		case <-me.done:
			// 此处必须调用，否则下次Init会有死锁
			me.over = true
			speaker.Clear()
			return
		case value := <-me.paused:
			speaker.Lock()
			me.ctrl.Paused = value
			speaker.Unlock()
		case <-me.increaseVol:
			speaker.Lock()
			me.volume.Volume += 0.5
			if me.volume.Volume > 0.001 {
				me.volume.Volume = 0
			}
			speaker.Unlock()
		case <-me.decreaseVol:
			speaker.Lock()
			me.volume.Volume -= 0.5
			speaker.Unlock()
		case <-time.After(time.Second):
			speaker.Lock()
			me.progress = float64(me.Streamer.Position())
			speaker.Unlock()
		}
	}
}

func (me *MusicEntry) Stop() {
	select {
	case me.done <- true:
	default:
	}
}
