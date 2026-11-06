package model

type IssueLabel struct {
	IssueID uint `gorm:"primaryKey"`
	LabelID uint `gorm:"primaryKey"`
}

func (IssueLabel) TableName() string {
	return "issue_labels"
}
