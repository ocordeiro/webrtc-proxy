package main

import (
	"context"
	"encoding/json"
	"github.com/centrifugal/centrifuge-go"
	"github.com/pion/webrtc/v3"
)
import "github.com/rs/zerolog/log"

type Message struct {
	Type string `json:"type"`
}

func main() {

	webRTCconfig := webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{
				URLs: []string{"stun:stun.l.google.com:19302"},
			},
		},
	}

	signaler := centrifuge.NewJsonClient(
		"ws://localhost:8000/connection/websocket",
		centrifuge.Config{
			Token: "eyJhbGciOiJSUzI1NiJ9.eyJzdWIiOiI0MyIsImV4cCI6MTY5OTk4NjYxMCwiaWF0IjoxNjY3Njg5OTc2fQ.vn9xdu5p5zWvlGRC0gneZZ3sr-mQhIfGhoNsQSaJ5c0MJXhMJtm1C1yc92ApYLiqjJhMnfs3EZH1bsWyDDco5QFhTG6ATslhGIdrGD29l1KV6Xnq5gSz22sUkruYORUGEj1EA873Au9DNgvCXIhfAh2OK3CNYG4iiGfgAeJ9KzLuB6iCE-5fs2-FubNAbaW-VPJUeCX4AIjk6C-OFy6f-QTjzTiQDtaLgkV5N0jGNvrM1kXsWUwh2Sgb_mKfObov0hKuF1Km6UfZcLGyeradIPqBiZXKkPeez1glNzZsK-FU9GkeX29o1la3gV5Hd_Tc6vxh0fcjXL9l_I8cAPO8-Q",
		},
	)

	err := signaler.Connect()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to signaler")
		return
	}

	peerConnection, err := webrtc.NewPeerConnection(webRTCconfig)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create peer connection")
	}

	dataChannel, err := peerConnection.CreateDataChannel("data", nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create data channel")
	}

	dataChannel.OnOpen(func() {
		log.Debug().Msg("Data channel opened")
	})

	dataChannel.OnClose(func() {
		log.Debug().Msg("Data channel closed")
	})

	peerConnection.OnICECandidate(func(candidate *webrtc.ICECandidate) {
		log.Debug().Msg("New ICE candidate provided")

		if candidate == nil {
			return
		}
		candidateData, err := json.Marshal(candidate.ToJSON())
		if err != nil {
			log.Error().Err(err).Msg("Failed to marshal ICE candidate")
			return
		}

		_, err = signaler.Publish(context.Background(), "offer", candidateData)
		if err != nil {
			log.Error().Err(err).Msg("Failed to publish ICE candidate")
			return
		}
	})

	offer, err := peerConnection.CreateOffer(nil)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create offer")
	}

	payload, err := json.Marshal(offer)
	if err != nil {
		log.Error().Err(err).Msg("Failed to marshal offer")
	}
	_, err = signaler.Publish(context.Background(), "offer", payload)

	err = peerConnection.SetLocalDescription(offer)
	if err != nil {
		log.Error().Err(err).Msg("Failed to set offer local description")
	}

	listen, err := signaler.NewSubscription("answer")
	if err != nil {
		log.Error().Err(err).Msg("Failed to create answer subscription")
	}

	listen.OnPublication(func(event centrifuge.PublicationEvent) {

		log.Debug().Msg("New publication received from answer")

		var message Message
		err := json.Unmarshal(event.Data, &message)
		if err != nil {
			log.Error().Err(err).Msg("Failed to unmarshal message")
			return
		}

		switch message.Type {
		case "answer":

			log.Debug().Msg("Received answer")

			var answer webrtc.SessionDescription
			err := json.Unmarshal(event.Data, &answer)
			if err != nil {
				log.Error().Err(err).Msg("Failed to unmarshal answer")
				return
			}

			err = peerConnection.SetRemoteDescription(answer)
			if err != nil {
				log.Error().Err(err).Msg("Failed to set answer remote answer description")
				return
			}
		default:
			log.Debug().Msg("Received candidate")

			var candidate webrtc.ICECandidateInit
			err = json.Unmarshal(event.Data, &candidate)
			if err != nil {
				log.Error().Err(err).Msg("Failed to unmarshal ICE candidate from answer")
				return
			}

			err := peerConnection.AddICECandidate(candidate)
			if err != nil {
				log.Error().Err(err).Msg("Failed to add ICE candidate from answer")
			}
		}
	})

	err = listen.Subscribe()
	if err != nil {
		log.Error().Err(err).Msg("Failed to subscribe to answer channel")
	}

	log.Info().Msg("Client started")

	select {}
}
