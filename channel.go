package tubemeta

import (
	"encoding/json"
	"fmt"
	"net/url"
	"regexp"
	"strings"
)

const (
	YOUTUBE_URL        = "https://www.youtube.com/channel/"
	YOUTUBE_CUSTOM_URL = "https://www.youtube.com/c/"
	YOUTUBE_USER_URL   = "https://www.youtube.com/"
)

type Channel struct {
	Id             string   `json:"id"`
	Name           string   `json:"name"`
	Description    string   `json:"description"`
	Avatar         string   `json:"avatar"`
	Banner         string   `json:"banner"`
	Url            string   `json:"url"`
	CustomUrl      string   `json:"custom_url"`
	Subscribers    string   `json:"subscribers"`
	Views          string   `json:"views"`
	CreatedAt      string   `json:"created_at"`
	Verified       bool     `json:"verified"`
	Live           bool     `json:"live"`
	Videos         string   `json:"videos"`
	Socials        []string `json:"socials"`
	OngoingStreams []string `json:"ongoing_streams"`
	Streams        []string `json:"streams"`
	CurrentStreams []string `json:"current_streams"`
}

type StreamData struct {
	AllStreams     []string `json:"all_streams"`
	OngoingStreams []string `json:"ongoing_streams"`
	CurrentStreams []string `json:"current_streams"`
}

type info struct {
	Metadata metadata `json:"metadata"`
}

type metadata struct {
	AboutChannelViewModel aboutChannelViewModel `json:"aboutChannelViewModel"`
}

type aboutChannelViewModel struct {
	ChannelId   string         `json:"channelId"`
	Description string         `json:"description"`
	CustomUrl   string         `json:"canonicalChannelUrl"`
	Subscribers string         `json:"subscriberCountText"`
	Views       string         `json:"viewCountText"`
	CreatedAt   joinedDateText `json:"joinedDateText"`
	Videos      string         `json:"videoCountText"`
}

type joinedDateText struct {
	Content string `json:"content"`
}

func getUrl(channelId string) (string, error) {
	re := regexp.MustCompile(`UC(.+)|c/(.+)|@(.+)`)
	result := re.FindStringSubmatch(channelId)

	if len(result) == 0 {
		return "", fmt.Errorf("invalid channel id")
	} else if result[1] != "" {
		return fmt.Sprintf("%sUC%s", YOUTUBE_URL, result[1]), nil
	} else if result[2] != "" {
		return fmt.Sprintf("%s%s", YOUTUBE_CUSTOM_URL, result[2]), nil
	} else if result[3] != "" {
		return fmt.Sprintf("%s@%s", YOUTUBE_USER_URL, result[3]), nil
	} else {
		return "", fmt.Errorf("invalid channel id")
	}
}

// GetChannel retrieves channel information from a given URL or channel ID.
// It returns a Channel struct containing the channel details and an error if any.
func GetChannel(channelId string) (*Channel, error) {
	url, err := getUrl(channelId)
	if err != nil {
		return nil, err
	}

	return getChannel(url)
}

func getChannel(url string) (*Channel, error) {
	content, err := getHtml(url + "/about")
	if err != nil || content == "" {
		fmt.Printf("error on getChannel: %s\n", err)
		return nil, err
	}

	channel := &Channel{}

	re, err := regexp.Compile(`\[{\"aboutChannelRenderer\":(.*?)}\],\"trackingParams`)
	if err != nil {
		return nil, err
	}

	var infoStr string

	infoTmp := re.FindStringSubmatch(content)
	for _, i := range infoTmp {
		if i != "" {
			infoStr = i
		}
	}

	if infoStr == "" {
		return nil, fmt.Errorf("channel not found")
	}

	info := &info{}
	err = json.Unmarshal([]byte(infoStr), info)
	if err != nil {
		return nil, err
	}
	channel.Id = info.Metadata.AboutChannelViewModel.ChannelId
	channel.Url = fmt.Sprintf("%s%s", YOUTUBE_URL, channel.Id)
	channel.Description = info.Metadata.AboutChannelViewModel.Description
	channel.CustomUrl = info.Metadata.AboutChannelViewModel.CustomUrl
	channel.Subscribers = info.Metadata.AboutChannelViewModel.Subscribers
	channel.Views = info.Metadata.AboutChannelViewModel.Views
	channel.CreatedAt = info.Metadata.AboutChannelViewModel.CreatedAt.Content
	channel.Videos = info.Metadata.AboutChannelViewModel.Videos
	channel.Name = parseName(content)
	channel.Avatar = parseAvatar(content)
	channel.Banner = parseBanner(content)
	channel.Live = checkLive(content)
	channel.Socials = parseSocials(content)
	channel.Verified = checkVerified(content)
	steamData, err := getStreamData(channel.Id)
	if err == nil {
		channel.OngoingStreams = steamData.OngoingStreams
		channel.Streams = steamData.AllStreams
		channel.CurrentStreams = steamData.CurrentStreams
	}

	return channel, nil
}

func parseName(content string) string {
	re, err := regexp.Compile(`channelMetadataRenderer\":{\"title\":\"(.*?)\"`)
	if err != nil {
		return ""
	}

	name := re.FindStringSubmatch(content)
	if len(name) == 0 {
		return ""
	}

	return name[1]
}

func parseAvatar(content string) string {
	re, err := regexp.Compile(`"height\":88},{\"url\":\"(.*?)\"`)
	if err != nil {
		return ""
	}

	avatar := re.FindStringSubmatch(content)
	if len(avatar) == 0 {
		return ""
	}

	return avatar[1]
}

func parseBanner(content string) string {
	re, err := regexp.Compile(`width\":1280,\"height\":351},{\"url\":\"(.*?)\"`)
	if err != nil {
		return ""
	}

	banner := re.FindStringSubmatch(content)
	if len(banner) == 0 {
		return ""
	}

	return banner[1]
}

func parseSocials(content string) []string {
	re, err := regexp.Compile(`q=https%3A%2F%2F(.*?)\"`)
	if err != nil {
		return nil
	}

	socialsMatch := re.FindAllStringSubmatch(content, -1)
	if len(socialsMatch) == 0 {
		return nil
	}
	socials := make([]string, len(socialsMatch))
	for i, s := range socialsMatch {
		url, err := url.QueryUnescape(s[1])
		if err != nil {
			continue
		}
		socials[i] = fmt.Sprintf("https://%s", url)
	}
	return removeDuplicate(socials)
}

func checkLive(content string) bool {
	re, err := regexp.Compile(`"style":"LIVE"`)
	if err != nil {
		return false
	}

	live := re.FindStringSubmatch(content)
	return len(live) != 0
}

func checkVerified(content string) bool {
	re, err := regexp.Compile(`{"text":"Verified"}`)
	if err != nil {
		return false
	}

	verified := re.FindStringSubmatch(content)
	return len(verified) != 0
}

func getStreamData(channelId string) (*StreamData, error) {
	url, err := getUrl(channelId)
	if err != nil {
		return nil, err
	}
	content, err := getHtml(url + "/streams")
	if err != nil || content == "" {
		return nil, err
	}

	allStreamIds, err := parseStreamIds(content)
	if err != nil {
		return nil, err
	}

	var (
		ongoingIds []string
		currentIds []string
	)
	for _, id := range allStreamIds {
		currentStream := fmt.Sprintf("vi/%s/hqdefault_live.jpg", id)
		if strings.Contains(content, currentStream) {
			currentIds = append(currentIds, id)
		}
	}
	ongoingIds, err = parseOngoingStreams(content)
	if err != nil {
		return nil, err
	}

	return &StreamData{
		AllStreams:     allStreamIds,
		OngoingStreams: ongoingIds,
		CurrentStreams: currentIds,
	}, nil
}

func parseStreamIds(content string) ([]string, error) {
	re, err := regexp.Compile(`"videoId":"(.*?)"`)
	if err != nil {
		return nil, err
	}

	streamIds := re.FindAllStringSubmatch(content, -1)
	if len(streamIds) == 0 {
		return nil, fmt.Errorf("stream not found")
	}

	var ids []string
	for _, id := range streamIds {
		ids = append(ids, id[1])
	}
	ids = removeDuplicate(ids)

	return ids, nil
}

func parseOngoingStreams(content string) ([]string, error) {
	re, err := regexp.Compile(`UPCOMING.*?addedVideoId\":\"(.*?)\"`)
	if err != nil {
		return nil, err
	}

	streamIds := re.FindAllStringSubmatch(content, -1)
	if len(streamIds) == 0 {
		return nil, fmt.Errorf("stream not found")
	}

	var ids []string
	for _, id := range streamIds {
		ids = append(ids, id[1])
	}
	ids = removeDuplicate(ids)

	return ids, nil
}
