package projects

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"sort"
	"time"
)

type DynamoStore struct {
	client *dynamodb.Client
	table  string
}

func NewDynamoStore(ctx context.Context, table, region, endpoint string) (*DynamoStore, error) {
	if table == "" || region == "" {
		return nil, fmt.Errorf("DYNAMODB_TABLE and AWS_REGION are required")
	}
	opts := []func(*config.LoadOptions) error{config.WithRegion(region)}
	if endpoint != "" {
		opts = append(opts, config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider("local", "local", "")))
	}
	cfg, err := config.LoadDefaultConfig(ctx, opts...)
	if err != nil {
		return nil, err
	}
	client := dynamodb.NewFromConfig(cfg, func(o *dynamodb.Options) {
		if endpoint != "" {
			o.BaseEndpoint = aws.String(endpoint)
		}
	})
	return &DynamoStore{client: client, table: table}, nil
}
func (s *DynamoStore) Mode() string { return "dynamodb" }
func (s *DynamoStore) Ready(ctx context.Context) error {
	_, err := s.client.DescribeTable(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(s.table)})
	return err
}
func (s *DynamoStore) EnsureLocalTable(ctx context.Context) error {
	err := s.Ready(ctx)
	var missing *types.ResourceNotFoundException
	if err == nil {
		return nil
	}
	if !errors.As(err, &missing) {
		return err
	}
	_, err = s.client.CreateTable(ctx, &dynamodb.CreateTableInput{TableName: aws.String(s.table), BillingMode: types.BillingModePayPerRequest, AttributeDefinitions: []types.AttributeDefinition{{AttributeName: aws.String("id"), AttributeType: types.ScalarAttributeTypeS}}, KeySchema: []types.KeySchemaElement{{AttributeName: aws.String("id"), KeyType: types.KeyTypeHash}}})
	if err != nil {
		return err
	}
	return dynamodb.NewTableExistsWaiter(s.client).Wait(ctx, &dynamodb.DescribeTableInput{TableName: aws.String(s.table)}, time.Minute)
}
func (s *DynamoStore) Put(ctx context.Context, p Project) error {
	item, err := attributevalue.MarshalMap(p)
	if err != nil {
		return err
	}
	_, err = s.client.PutItem(ctx, &dynamodb.PutItemInput{TableName: aws.String(s.table), Item: item, ConditionExpression: aws.String("attribute_not_exists(id)")})
	var conflict *types.ConditionalCheckFailedException
	if errors.As(err, &conflict) {
		return ErrConflict
	}
	return err
}
func (s *DynamoStore) Get(ctx context.Context, id string) (Project, error) {
	result, err := s.client.GetItem(ctx, &dynamodb.GetItemInput{TableName: aws.String(s.table), Key: map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: id}}, ConsistentRead: aws.Bool(true)})
	if err != nil {
		return Project{}, err
	}
	if len(result.Item) == 0 {
		return Project{}, ErrNotFound
	}
	var p Project
	err = attributevalue.UnmarshalMap(result.Item, &p)
	return p, err
}
func (s *DynamoStore) List(ctx context.Context, limit int, cursor string) (Page, error) {
	input := &dynamodb.ScanInput{TableName: aws.String(s.table), Limit: aws.Int32(int32(limit)), ConsistentRead: aws.Bool(true)}
	if cursor != "" {
		id, err := base64.RawURLEncoding.DecodeString(cursor)
		if err != nil || len(id) == 0 {
			return Page{}, ErrCursor
		}
		input.ExclusiveStartKey = map[string]types.AttributeValue{"id": &types.AttributeValueMemberS{Value: string(id)}}
	}
	result, err := s.client.Scan(ctx, input)
	if err != nil {
		return Page{}, err
	}
	page := Page{Projects: []Project{}}
	if err = attributevalue.UnmarshalListOfMaps(result.Items, &page.Projects); err != nil {
		return Page{}, err
	}
	sort.Slice(page.Projects, func(i, j int) bool { return page.Projects[i].CreatedAt > page.Projects[j].CreatedAt })
	if id, ok := result.LastEvaluatedKey["id"].(*types.AttributeValueMemberS); ok {
		page.NextCursor = base64.RawURLEncoding.EncodeToString([]byte(id.Value))
	}
	return page, nil
}
