package tubemeta

import (
	"encoding/json"
	"fmt"
	"html"
	"regexp"
	"strconv"
	"strings"
	"time"
)

type Video struct {
	Id                 string
	Title              string
	Description        string
	ViewCount          int
	LikeCount          int
	LiveNow            bool
	LiveContent        bool
	ScheduledStartTime time.Time
	UploadDate         time.Time
	Thumbnails         []string
	URL                string
	Tags               []string
	Duration           string
	ChannelId          string
	Genre              string
}

type videoMetadata struct {
	VideoId          string `json:"videoId"`
	Title            string `json:"title"`
	ShortDescription string `json:"shortDescription"`
	IsLiveContent    bool   `json:"isLiveContent"`
	ViewCount        string `json:"viewCount"`
	Duration         string `json:"lengthSeconds"`
	Thumbnail        struct {
		Thumbnails []struct {
			URL    string `json:"url"`
			Width  int    `json:"width"`
			Height int    `json:"height"`
		} `json:"thumbnails"`
	} `json:"thumbnail"`
	ChannelId string   `json:"channelId"`
	Tags      []string `json:"keywords"`
}

type liveBroadcastDetails struct {
	IsLiveNow      bool   `json:"isLiveNow"`
	StartTimestamp string `json:"startTimestamp"`
}

func GetVideo(urlOrId string) (video Video, err error) {
	regex, err := regexp.Compile(`.be\/([A-Za-z\d_-]{11})|v\=([A-Za-z\d_-]{11})$|^([A-Za-z\d_-]{11})$`)
	if err != nil {
		return
	}
	idMatch := regex.FindStringSubmatch(urlOrId)
	if len(idMatch) == 0 {
		err = ErrInvalidVideoId
		return
	}
	// skip first match
	for _, match := range idMatch[1:] {
		if match != "" {
			video.Id = match
		}
	}
	if video.Id == "" {
		err = ErrInvalidVideoId
		return
	}

	video.URL = "https://www.youtube.com/watch?v=" + video.Id
	content, err := getHtml(video.URL)
	if err != nil {
		err = ErrInvalidUrl
		return
	}

	// regex patterns
	detailsPattern := regexp.MustCompile(`videoDetails":(.*?)"isLiveContent":.*?}`)
	uploadDatePattern := regexp.MustCompile("<meta itemprop=\"uploadDate\" content=\"(.*?)\">")
	genrePattern := regexp.MustCompile("<meta itemprop=\"genre\" content=\"(.*?)\">")
	likeCountPattern := regexp.MustCompile(`expandedLikeCountIfIndifferent":{"content":"(.*?)"}`)
	liveBroadcastDetailsPattern := regexp.MustCompile("liveBroadcastDetails\":\\{(.*?)\\}")

	uploadDateMatch := uploadDatePattern.FindStringSubmatch(content)
	if len(uploadDateMatch) > 0 {
		dateTime, err := time.Parse(time.RFC3339, uploadDateMatch[1])
		if err == nil {
			video.UploadDate = dateTime
		}
	}

	video.LikeCount = 0
	likeCountMatch := likeCountPattern.FindStringSubmatch(content)
	if len(likeCountMatch) > 0 {
		likeCountMatch[1] = strings.Replace(likeCountMatch[1], ",", "", -1)
		likeCount, err := strconv.Atoi(likeCountMatch[1])
		if err == nil {
			video.LikeCount = likeCount
		}
	}

	liveBroadcastDetailsMatch := liveBroadcastDetailsPattern.FindStringSubmatch(content)
	if len(liveBroadcastDetailsMatch) > 0 {
		liveBroadcastDetailsJson := fmt.Sprintf("{%s}", liveBroadcastDetailsMatch[1])
		liveBroadcastDetails := &liveBroadcastDetails{}
		json.Unmarshal([]byte(liveBroadcastDetailsJson), liveBroadcastDetails)

		video.LiveNow = liveBroadcastDetails.IsLiveNow
		if liveBroadcastDetails.StartTimestamp != "" {
			startTime, err := time.Parse(time.RFC3339, liveBroadcastDetails.StartTimestamp)
			if err == nil {
				video.ScheduledStartTime = startTime
			}
		}
	}

	rawDetailsMatch := detailsPattern.FindStringSubmatch(content)
	if len(rawDetailsMatch) == 0 {
		err = ErrInvalidUrl
		return
	}
	rawDetails := strings.Replace(rawDetailsMatch[0], "videoDetails\":", "", 1)
	metadata := &videoMetadata{}
	err = json.Unmarshal([]byte(rawDetails), metadata)
	if err != nil {
		err = ErrInvalidUrl
		return
	}

	video.Title = metadata.Title
	video.Description = metadata.ShortDescription
	video.ViewCount = 0
	video.LiveContent = metadata.IsLiveContent
	video.Duration = metadata.Duration
	video.ChannelId = metadata.ChannelId
	video.Thumbnails = make([]string, len(metadata.Thumbnail.Thumbnails))
	for i, thumbnail := range metadata.Thumbnail.Thumbnails {
		video.Thumbnails[i] = thumbnail.URL
	}
	video.Tags = metadata.Tags
	viewCount, err := strconv.Atoi(metadata.ViewCount)
	if err == nil {
		video.ViewCount = viewCount
	}

	genreMatch := genrePattern.FindStringSubmatch(content)
	if len(genreMatch) > 0 {
		video.Genre = html.UnescapeString(genreMatch[1])
	}

	return
}
