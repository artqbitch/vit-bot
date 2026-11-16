package base

import (
	"context"

	"cloud.google.com/go/firestore"
	firebase "firebase.google.com/go/v4"
	"google.golang.org/api/iterator"
	"google.golang.org/api/option"
)

type AnswerBox struct {
	ID       string `firestore:"-"`
	Question string `firestore:"question"`
	Answer   string `firestore:"answer"`
}

type Base struct {
	Client *firestore.Client
}

func Connect(ctx context.Context) (*Base, error) {
	app, err := firebase.NewApp(ctx, nil, option.WithAuthCredentialsFile(option.ServiceAccount, "serviceAccount.json"))

	if err != nil {
		return nil, err
	}

	db, err := app.Firestore(ctx)
	if err != nil {
		return nil, err
	}
	return &Base{Client: db}, nil
}

func (db *Base) GetBoxes(ctx context.Context) ([]AnswerBox, error) {
	query := db.Client.Collection("boxes").Documents(ctx)

	defer query.Stop()

	var boxes []AnswerBox

	for {
		doc, err := query.Next()

		if err == iterator.Done {
			break
		}

		if err != nil {
			return nil, err
		}

		var b AnswerBox

		if err := doc.DataTo(&b); err != nil {
			return nil, err
		}
		b.ID = doc.Ref.ID
		boxes = append(boxes, b)
	}

	return boxes, nil
}
