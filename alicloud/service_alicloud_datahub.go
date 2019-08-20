// common functions used by datahub
package alicloud

import (
	"strings"
	"time"

	"github.com/terraform-providers/terraform-provider-alicloud/alicloud/connectivity"

	"github.com/aliyun/aliyun-datahub-sdk-go/datahub"
)

type DatahubService struct {
	client *connectivity.AliyunClient
}

func (s *DatahubService) DescribeDatahubProject(id string) (project *datahub.Project, err error) {
	var requestInfo *datahub.DataHub

	raw, err := s.client.WithDataHubClient(func(dataHubClient *datahub.DataHub) (interface{}, error) {
		requestInfo = dataHubClient
		return dataHubClient.GetProject(id)
	})
	if err != nil {
		if isDatahubNotExistError(err) {
			return project, WrapErrorf(err, NotFoundMsg, AliyunDatahubSdkGo)
		}
		return nil, WrapErrorf(err, DefaultErrorMsg, id, "GetProject", AliyunDatahubSdkGo)
	}
	if debugOn() {
		requestMap := make(map[string]string)
		requestMap["ProjectName"] = id
		addDebug("GetProject", raw, requestInfo, requestMap)
	}
	project, _ = raw.(*datahub.Project)
	if project == nil {
		return project, WrapErrorf(Error(GetNotFoundMessage("DatahubProject", id)), NotFoundMsg, ProviderERROR)
	}
	return
}

func (s *DatahubService) WaitForDatahubProject(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)

	for {
		object, err := s.DescribeDatahubProject(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}

		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, object.String(), id, ProviderERROR)
		}

	}
}

func (s *DatahubService) DescribeDatahubSubscription(id string) (subscription *datahub.Subscription, err error) {
	parts, err := ParseResourceId(id, 3)
	if err != nil {
		return nil, WrapError(err)
	}
	projectName, topicName, subId := parts[0], parts[1], parts[2]

	var requestInfo *datahub.DataHub

	raw, err := s.client.WithDataHubClient(func(dataHubClient *datahub.DataHub) (interface{}, error) {
		requestInfo = dataHubClient
		return dataHubClient.GetSubscription(projectName, topicName, subId)
	})
	if err != nil {
		if isDatahubNotExistError(err) {
			return subscription, WrapErrorf(err, NotFoundMsg, AliyunDatahubSdkGo)
		}
		return nil, WrapErrorf(err, DefaultErrorMsg, id, "GetSubscription", AliyunDatahubSdkGo)
	}
	if debugOn() {
		requestMap := make(map[string]string)
		requestMap["ProjectName"] = projectName
		requestMap["TopicName"] = topicName
		requestMap["SubId"] = subId
		addDebug("GetProject", raw, requestInfo, requestMap)
	}
	subscription, _ = raw.(*datahub.Subscription)
	if subscription == nil || subscription.TopicName != topicName || subscription.SubId != subId {
		return subscription, WrapErrorf(Error(GetNotFoundMessage("DatahubSubscription", id)), NotFoundMsg, ProviderERROR)
	}
	return
}

func (s *DatahubService) WaitForDatahubSubscription(id string, status Status, timeout int) error {
	deadline := time.Now().Add(time.Duration(timeout) * time.Second)
	parts, err := ParseResourceId(id, 3)
	if err != nil {
		return WrapError(err)
	}
	topicName, subId := parts[1], parts[2]
	for {
		object, err := s.DescribeDatahubSubscription(id)
		if err != nil {
			if NotFoundError(err) {
				if status == Deleted {
					return nil
				}
			} else {
				return WrapError(err)
			}
		}
		if object.TopicName == topicName && object.SubId == subId && status != Deleted {
			return nil
		}
		if time.Now().After(deadline) {
			return WrapErrorf(err, WaitTimeoutMsg, id, GetFunc(1), timeout, object.TopicName+":"+object.SubId, parts[1]+":"+parts[2], ProviderERROR)
		}

	}
}

func convUint64ToDate(t uint64) string {
	return time.Unix(int64(t), 0).Format("2006-01-02 15:04:05")
}

func getNow() string {
	return time.Now().Format("2006-01-02 15:04:05")
}

func getRecordSchema(typeMap map[string]interface{}) (recordSchema *datahub.RecordSchema) {
	recordSchema = datahub.NewRecordSchema()

	for k, v := range typeMap {
		recordSchema.AddField(datahub.Field{Name: string(k), Type: datahub.FieldType(v.(string))})
	}

	return recordSchema
}

func isRetryableDatahubError(err error) bool {
	if e, ok := err.(datahub.DatahubError); ok && e.StatusCode >= 500 {
		return true
	}

	return false
}

// It is proactive defense to the case that SDK extends new datahub objects.
const (
	DoesNotExist = "does not exist"
)

func isDatahubNotExistError(err error) bool {
	return IsExceptedErrors(err, []string{datahub.NoSuchProject, datahub.NoSuchTopic, datahub.NoSuchShard, datahub.NoSuchSubscription, DoesNotExist})
}

func isTerraformTestingDatahubObject(name string) bool {
	prefixes := []string{
		"tf_testAcc",
		"tf_test_",
		"testAcc",
	}

	for _, prefix := range prefixes {
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(prefix)) {
			return true
		}
	}

	return false
}

func getDefaultRecordSchemainMap() map[string]interface{} {

	return map[string]interface{}{
		"string_field": "STRING",
	}
}
