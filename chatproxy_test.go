package chatproxy_test

import (
	"github.com/mr-joshcrane/chatproxy"
	"testing"
  "os"
  "fmt"
  "strconv"
)

func TestInterface(t *testing.T) {
	t.Parallel()

	cases := []struct {
		query string
		want  int
	}{
		{
			"I want to find a book about animals.",
			7,
		},
		{
			"I'm interested in cooking.",
			2,
		},
		{
			"I'm interested in how we used to live.",
			20,
		},
		{
			"I want to find a book about COBOL",
			0,
		},
    {
      "I'm interested in evolution",
      0,
    },
	}

	database := map[int]string{
		1: "Quantum Computing: A Leap Forward",
		2: "Mastering the Art of French Cooking",
		3: "Exploring the Depths: Oceanography Today",
		4: "Bach: The Composer Who Transformed Music",
		5: "The Psychology of Laughter",
		6: "The World of Abstract Art",
		7: "The Secret Lives of Ants",
		8: "The Impact of Social Media on Society",
		9: "The Future of Renewable Energy",
		10: "Origami: The Art of Paper Folding",
		11: "Understanding the Universe: The Latest in Astrophysics",
		12: "The History of Cinema",
		13: "Exploring the World's Most Extreme Environments",
		14: "The Changing Landscape of Sports",
		15: "Fashion Through the Decades",
		16: "The Role of Mythology in Modern Culture",
		17: "The Wonders of Robotics",
		18: "The Importance of Mental Health",
		19: "The Evolving World of Comic Books",
		20: "The Mysteries of Ancient Civilizations",
	}
	token := os.Getenv("OPENAPI_TOKEN")
	client, err := chatproxy.NewChatGPTClient(token)
  if err != nil {
    t.Errorf("Error: %s", err)
  }

	client.SetPurpose(
		` I will ask you to find me (at most) one record from the library based on my query.
      The library will be provided to you in the form of a Golang struct.
      You will respond ONLY with the key (an integer) of the record that matches the query.
      If no records match the query, you will return the integer '0'
    `,
	)
	client.SetPurpose(fmt.Sprintf("The library is as follows: %v", database))

	for _, testCase := range cases {
		client.RecordMessage(chatproxy.ChatMessage{
			Content: testCase.query,
			Role:    chatproxy.RoleUser,
		})
		message, err := client.GetCompletion()
		if err != nil {
			t.Errorf("Error: %s", err)
		}
    got, err := strconv.Atoi(message)
    if err != nil {
      t.Errorf("Error: %s: %s", err, message)
    }
		if got != testCase.want {
			t.Errorf("Query: %s, Got: %d, want: %d", testCase.query, got, testCase.want)
		}
	}
}
