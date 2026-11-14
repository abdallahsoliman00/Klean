package store

import "database/sql"

// Image is a picture attached to a message, as the browser sees it. The
// bytes are served separately, by id.
type Image struct {
	ID        string `json:"id"`
	MediaType string `json:"mediaType"`
}

// NewImage is a picture to attach to a message being added.
type NewImage struct {
	MediaType string
	Data      []byte
}

// ImageData is a stored picture's bytes and their type.
type ImageData struct {
	MediaType string
	Data      []byte
}

func insertImages(tx *sql.Tx, messageID string, images []NewImage) error {
	for i, img := range images {
		if _, err := tx.Exec("INSERT INTO images (id, message_id, media_type, data, position) VALUES (?,?,?,?,?)",
			newID(), messageID, img.MediaType, img.Data, i); err != nil {
			return err
		}
	}
	return nil
}

// queryImages lists the images matching where (over images i), in order, by
// the id of their message.
func queryImages(q queryer, where string, args ...any) (map[string][]Image, error) {
	rows, err := q.Query("SELECT i.message_id, i.id, i.media_type FROM images i WHERE "+where+" ORDER BY i.position", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]Image{}
	for rows.Next() {
		var messageID string
		var img Image
		if err := rows.Scan(&messageID, &img.ID, &img.MediaType); err != nil {
			return nil, err
		}
		out[messageID] = append(out[messageID], img)
	}
	return out, rows.Err()
}

// MessageImages lists the images of one message.
func (d *DB) MessageImages(messageID string) ([]Image, error) {
	images, err := queryImages(d.sql, "i.message_id = ?", messageID)
	return images[messageID], err
}

// GetImage returns nil, nil when there is no such image.
func (d *DB) GetImage(id string) (*ImageData, error) {
	var img ImageData
	err := d.sql.QueryRow("SELECT media_type, data FROM images WHERE id = ?", id).Scan(&img.MediaType, &img.Data)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	return &img, err
}

// ChatImageData returns the bytes of every image in a chat, by image id.
// shortcut: reads all of a chat's images into memory for each reply, also
// those the context window will leave out; load per kept turn if chats with
// many large images make replies slow.
func (d *DB) ChatImageData(chatID string) (map[string][]byte, error) {
	rows, err := d.sql.Query(`SELECT i.id, i.data FROM images i
		JOIN messages m ON m.id = i.message_id WHERE m.chat_id = ?`, chatID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := map[string][]byte{}
	for rows.Next() {
		var id string
		var data []byte
		if err := rows.Scan(&id, &data); err != nil {
			return nil, err
		}
		out[id] = data
	}
	return out, rows.Err()
}
