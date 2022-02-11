package torrentcfg

import (
	"time"

	lg "github.com/anacrolix/log"
	"github.com/anacrolix/torrent"
	"github.com/anacrolix/torrent/storage"
	"github.com/c2h5oh/datasize"
	"golang.org/x/time/rate"
)

// DefaultPieceSize - Erigon serves many big files, bigger pieces will reduce
// amount of network announcements, but can't go over 2Mb
// see https://wiki.theory.org/BitTorrentSpecification#Metainfo_File_Structure
const DefaultPieceSize = 2 * 1024 * 1024

func Default() *torrent.ClientConfig {
	torrentConfig := torrent.NewDefaultClientConfig()

	// enable dht
	torrentConfig.NoDHT = true
	//torrentConfig.DisableTrackers = true
	//torrentConfig.DisableWebtorrent = true
	//torrentConfig.DisableWebseeds = true

	// Increase default timeouts, because we often run on commodity networks
	torrentConfig.MinDialTimeout = 6 * time.Second      // default: 3sec
	torrentConfig.NominalDialTimeout = 20 * time.Second // default: 20sec
	torrentConfig.HandshakesTimeout = 8 * time.Second   // default: 4sec

	// We would-like to reduce amount of goroutines in Erigon, so reducing next params
	torrentConfig.EstablishedConnsPerTorrent = 5 // default: 50
	torrentConfig.TorrentPeersHighWater = 10     // default: 500
	torrentConfig.TorrentPeersLowWater = 5       // default: 50
	torrentConfig.HalfOpenConnsPerTorrent = 5    // default: 25
	torrentConfig.TotalHalfOpenConns = 10        // default: 100

	return torrentConfig
}

func New(snapshotsDir string, verbosity lg.Level, downloadRate, uploadRate datasize.ByteSize, torrentPort int) (*torrent.ClientConfig, error) {
	torrentConfig := Default()
	torrentConfig.ListenPort = torrentPort
	torrentConfig.Seed = true
	torrentConfig.DataDir = snapshotsDir
	torrentConfig.UpnpID = torrentConfig.UpnpID + "leecher"

	// rates are divided by 2 - I don't know why it works, maybe bug inside torrent lib accounting
	torrentConfig.UploadRateLimiter = rate.NewLimiter(rate.Limit(uploadRate.Bytes()/2), 2*DefaultPieceSize)     // default: unlimited
	torrentConfig.DownloadRateLimiter = rate.NewLimiter(rate.Limit(downloadRate.Bytes()/2), 2*DefaultPieceSize) // default: unlimited

	// debug
	if lg.Debug == verbosity {
		torrentConfig.Debug = true
	}
	torrentConfig.Logger = NewAdapterLogger().FilterLevel(verbosity)

	//c, err := storage.NewBoltPieceCompletion(snapshotsDir)
	//if err != nil {
	//	return nil, err
	//}
	torrentConfig.DefaultStorage = storage.NewFile(snapshotsDir)
	return torrentConfig, nil
}
