package message

import (
	"testing"
)

var (
	msgs = []Message{{
		MessageNode{
			Props: Properties{
				EnableList:  EMPropBold,
				InheritList: EMPropAll,
			},
			Text: "testing",
		},
		MessageNode{
			Props: Properties{
				EnableList:  EMPropItalic,
				InheritList: EMPropAll,
			},
			Text: "123",
		},
	}}
)

var testMessages = []Message{
	Message{
		MessageNode{
			Props: Properties{EnableList: EMPropBold, InheritList: EMPropAll},
			Text:  "testing",
		},
		MessageNode{
			Props: Properties{EnableList: EMPropItalic, InheritList: EMPropBold},
			Text:  "123",
		},
	},
}

type flattenTest struct {
	expectedProps []PropertyList
}

type toIntermediateTest struct {
	expectedStr string
}

type fullMessageTest struct {
	message            Message
	flattenTest        flattenTest
	toIntermediateTest toIntermediateTest
}

var (
	messageTests = []fullMessageTest{
		{
			message: testMessages[0],
			flattenTest: flattenTest{expectedProps: []PropertyList{
				EMPropBold,
				EMPropItalic | EMPropBold,
			}},
			toIntermediateTest: toIntermediateTest{expectedStr: "1:63:7:testing2:1:3:123"},
		},
	}
)

type fromIntermediateTest struct {
	flattened bool //whether to check only flattened messages
	from      string
	expected  Message
}

var fromIntermediateTests = []fromIntermediateTest{
	{
		flattened: false,
		from:      "1:63:7:testing2:1:-1:123",
		expected:  testMessages[0],
	},
}

func TestMessage_Flatten(t *testing.T) {
	for nTest, test := range messageTests {
		flat := test.message.Flatten()

		if len(flat) != 2 {
			t.Errorf("(test %d) Length of the flattened message is not that of the original message (%d (flat) != %d (original))",
				nTest, len(flat), len(test.message))
		}

		for k, v := range flat {
			if v.Props.InheritList != 0 {
				t.Errorf("(test %d) Nonzero inherit list at index %d: %d",
					nTest, k, v.Props.InheritList)
			}
		}

		for k, v := range flat {
			if v.Props.EnableList != test.flattenTest.expectedProps[k] {
				t.Errorf("(test %d) Invalid enable list at index %d: expected %d, got %d",
					nTest, k, test.flattenTest.expectedProps[k], v.Props.EnableList)
			}
		}
	}
}

func TestMessage_ToIntermediate(t *testing.T) {
	for nTest, test := range messageTests {
		str := test.message.ToIntermediate()
		if str != test.toIntermediateTest.expectedStr {
			t.Errorf("(test %d) Bad intermediate string: expected \"%s\", got \"%s\"",
				nTest, test.toIntermediateTest.expectedStr, str)
		}
	}
}

func TestFromIntermediate(t *testing.T) {
	for nTest, test := range fromIntermediateTests {
		got, err := FromIntermediate(test.from)
		if err != nil {
			t.Errorf("(test %d) Failed with error: %s", nTest, err)
		}

		b := false
		if test.flattened {
			b = got.VisiblyEquals(test.expected)
		} else {
			b = got.StrictlyEquals(test.expected)
		}

		if !b {
			t.Errorf("(test %d) Failed comparison", nTest)
		}
	}
}
