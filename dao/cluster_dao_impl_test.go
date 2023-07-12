package dao

import (
	"errors"
	"github.com/golang/mock/gomock"
	e "github.com/myntra/goscheduler/cluster_entity"
	"github.com/myntra/goscheduler/conf"
	"github.com/myntra/goscheduler/mocks"
	"github.com/myntra/goscheduler/store"
	"sync"
	"testing"
)

func setupClusterDaoMocks(t *testing.T) (*ClusterDaoImplCassandra, *mocks.MockSessionInterface, *mocks.MockQueryInterface, *mocks.MockIterInterface, *gomock.Controller) {
	dao := &ClusterDaoImplCassandra{
		ClusterConfig: &conf.ClusterConfig{
			PageSize: 10,
			NumRetry: 2,
		},
		ClusterDBConfig: &conf.ClusterDBConfig{
			ClusterKeySpace:   "",
			DBConfig:          conf.CassandraConfig{},
			EntityHistorySize: 0,
		},
		AppMap: AppMap{
			lock: sync.RWMutex{},
			m:    make(map[string]store.App),
		},
	}

	ctrl := gomock.NewController(t)
	m := mocks.NewMockSessionInterface(ctrl)
	mq := mocks.NewMockQueryInterface(ctrl)
	mItr := mocks.NewMockIterInterface(ctrl)

	dao.Session = m

	return dao, m, mq, mItr, ctrl
}

func TestClusterDaoImplCassandra_GetAllEntitiesInfoOfNode(t *testing.T) {
	dao, m, mq, mItr, ctrl := setupClusterDaoMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Consistency(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().PageSize(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Iter().Return(mItr).AnyTimes()
	mItr.EXPECT().Close().Return(nil).AnyTimes()

	for _, test := range []struct {
		Input    []*gomock.Call
		Expected int
	}{
		{
			[]*gomock.Call{
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Tony").
					Return(true).
					Times(1),
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Steve").
					Return(true).
					Times(1),
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Thor").
					Return(false).
					Times(1),
			},
			2,
		},
		{
			[]*gomock.Call{
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "SRK").
					Return(true).
					Times(2),
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Ranbir").
					Return(true).
					Times(1),
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Alia").
					Return(false).
					Times(1),
			},
			3,
		},
		{
			[]*gomock.Call{
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "No One").
					Return(false).
					Times(1),
			},
			0,
		},
	} {
		gomock.InOrder(test.Input...)
		if entities := dao.GetAllEntitiesInfoOfNode("test"); len(entities) != test.Expected {
			t.Errorf("Got length: %d, expected: %d", len(entities), test.Expected)
		}
	}
}

func TestClusterDaoImplCassandra_GetAllEntitiesInfo(t *testing.T) {
	dao, m, mq, mItr, ctrl := setupClusterDaoMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Consistency(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().PageSize(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Iter().Return(mItr).AnyTimes()
	mItr.EXPECT().Close().Return(nil).AnyTimes()

	for _, test := range []struct {
		Input    []*gomock.Call
		Expected int
	}{
		{
			[]*gomock.Call{
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Tony").
					Return(true).
					Times(1),
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Steve").
					Return(true).
					Times(1),
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Thor").
					Return(false).
					Times(1),
			},
			2,
		},
		{
			[]*gomock.Call{
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Dr. Strange").
					Return(true).
					Times(2),
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Wanda").
					Return(true).
					Times(1),
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Vision").
					Return(false).
					Times(1),
			},
			3,
		},
		{
			[]*gomock.Call{
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Thanos").
					Return(false).
					Times(1),
			},
			0,
		},
	} {
		gomock.InOrder(test.Input...)
		if entities := dao.GetAllEntitiesInfo(); len(entities) != test.Expected {
			t.Errorf("Got length: %d, expected: %d", len(entities), test.Expected)
		}
	}
}

func TestClusterDaoImplCassandra_GetEntityInfo(t *testing.T) {
	dao, m, mq, _, ctrl := setupClusterDaoMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Consistency(gomock.Any()).Return(mq).AnyTimes()

	// GetEntityInfo success
	for _, test := range []struct {
		Input    *gomock.Call
		Expected string
	}{
		{
			mq.
				EXPECT().
				Scan(gomock.All()).
				SetArg(0, "Tony").
				Return(nil).
				Times(1),
			"Tony",
		},
		{
			mq.
				EXPECT().
				Scan(gomock.All()).
				SetArg(0, "Steve").
				Return(nil).
				Times(1),
			"Steve",
		},
		{
			mq.
				EXPECT().
				Scan(gomock.All()).
				SetArg(0, "").
				Return(nil).
				Times(1),
			"",
		},
	} {
		if entity := dao.GetEntityInfo(test.Expected); entity.Id != test.Expected {
			t.Errorf("Got id: %s, expected: %s", entity.Id, test.Expected)
		}
	}

	// GetEntityInfo failure
	for _, test := range []struct {
		Input    []interface{}
		Expected e.EntityInfo
	}{
		{
			[]interface{}{
				"Tony",
				mq.
					EXPECT().
					Scan(gomock.All()).
					Return(errors.New("no result found for Steve")).
					Times(1),
			},
			e.EntityInfo{},
		},
		{
			[]interface{}{
				"Steve",
				mq.
					EXPECT().
					Scan(gomock.All()).
					Return(errors.New("no result found for Steve")).
					Times(1),
			},
			e.EntityInfo{},
		},
		{
			[]interface{}{
				"Thor",
				mq.
					EXPECT().
					Scan(gomock.All()).
					Return(errors.New("no result found for Thor")).
					Times(1),
			},
			e.EntityInfo{},
		},
	} {
		if entity := dao.GetEntityInfo(test.Input[0].(string)); entity != test.Expected {
			t.Errorf("Got entity: %+v, expected: %+v", entity, test.Expected)
		}
	}
}

func TestClusterDaoImplCassandra_InsertApp(t *testing.T) {
	dao, m, mq, _, ctrl := setupClusterDaoMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mq).AnyTimes()
	gomock.InOrder(
		mq.EXPECT().Exec().Return(nil).Times(2),
		mq.EXPECT().Exec().Return(errors.New("error inserting app")).Times(2),
	)

	// InsertApp success
	for _, test := range []struct {
		Input    store.App
		Expected error
	}{
		{
			store.App{
				AppId:      "Tony$123",
				Partitions: 5,
				Active:     true,
			},
			nil,
		},
		{
			store.App{
				AppId:      "#_Steve",
				Partitions: 5,
				Active:     true,
			},
			nil,
		},
	} {
		if err := dao.InsertApp(test.Input); err != test.Expected {
			t.Errorf("Got error: %+v, expected: %+v", err, test.Expected)
		}
	}

	// InsertApp failure
	for _, test := range []struct {
		Input    store.App
		Expected error
	}{
		{
			store.App{
				AppId:      "Tony$123",
				Partitions: 5,
				Active:     true,
			},
			errors.New("error inserting app"),
		},
		{
			store.App{
				AppId:      "#_Steve",
				Partitions: 5,
				Active:     true,
			},
			errors.New("error inserting app"),
		},
	} {
		if err := dao.InsertApp(test.Input); err.Error() != test.Expected.Error() {
			t.Errorf("Got error: %+v, expected: %+v", err, test.Expected)
		}
	}
}

func TestClusterDaoImplCassandra_GetApp(t *testing.T) {
	dao, m, mq, _, ctrl := setupClusterDaoMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Consistency(gomock.Any()).Return(mq).AnyTimes()

	// GetApp success
	for _, test := range []struct {
		Input    []interface{}
		Expected store.App
	}{
		{
			[]interface{}{
				"Tony",
				mq.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Tony").
					Return(nil).
					Times(1),
			},
			store.App{
				AppId:      "Tony",
				Partitions: 0,
				Active:     false,
			},
		},
		{
			[]interface{}{
				"Steve",
				mq.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Steve").
					Return(nil).
					Times(1),
			},
			store.App{
				AppId:      "Steve",
				Partitions: 0,
				Active:     false,
			},
		},
		{
			[]interface{}{
				"Thor",
				mq.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Thor").
					Return(nil).
					Times(1),
			},
			store.App{
				AppId:      "Thor",
				Partitions: 0,
				Active:     false,
			},
		},
	} {
		if app, _ := dao.GetApp(test.Input[0].(string)); app.AppId != test.Expected.AppId {
			t.Errorf("Got entity: %+v, expected: %+v", app, test.Expected)
		}
	}

	//Empty the map to avoid getting from cache
	dao.AppMap.m = map[string]store.App{}

	// GetApp failure
	for _, test := range []struct {
		Input    []interface{}
		Expected error
	}{
		{
			[]interface{}{
				"SRK",
				mq.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "SRK").
					Return(errors.New("error getting app SRK")).
					Times(1),
			},
			errors.New("error getting app SRK"),
		},
		{
			[]interface{}{
				"Ranbir",
				mq.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Ranbir").
					Return(errors.New("error getting app Ranbir")).
					Times(1),
			},
			errors.New("error getting app Ranbir"),
		},
	} {
		if _, err := dao.GetApp(test.Input[0].(string)); err.Error() != test.Expected.Error() {
			t.Errorf("Got error: %+v, expected: %+v", err, test.Expected)
		}
	}
}

func TestClusterDaoImplCassandra_UpdateEntityStatus(t *testing.T) {
	dao, m, mq, _, ctrl := setupClusterDaoMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Consistency(gomock.Any()).Return(mq).AnyTimes()

	for _, test := range []struct {
		Input    []interface{}
		Mock     []*gomock.Call
		Expected error
	}{
		{
			[]interface{}{
				"Tony",
				"localhost",
				1,
			},
			[]*gomock.Call{
				mq.
					EXPECT().
					Scan(gomock.All()).
					Return(nil).
					Times(1),
				mq.
					EXPECT().
					Exec().
					Return(nil).
					Times(1),
			},
			nil,
		},
		{
			[]interface{}{
				"Steve",
				"localhost",
				0,
			},
			[]*gomock.Call{
				mq.
					EXPECT().
					Scan(gomock.All()).
					Return(errors.New("error getting app")).
					Times(1),
				mq.
					EXPECT().
					Exec().
					Return(errors.New("error updating app")).
					Times(1),
			},
			errors.New("error updating app"),
		},
	} {
		if err := dao.UpdateEntityStatus(test.Input[0].(string), test.Input[1].(string), test.Input[2].(int)); err != nil && err.Error() != test.Expected.Error() {
			t.Errorf("Got error: %+v, expected: %+v", err, test.Expected)
		}
	}
}

func TestClusterDaoImplCassandra_CreateEntity(t *testing.T) {
	dao, m, mq, _, ctrl := setupClusterDaoMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(mq).AnyTimes()

	// GetEntityInfo success
	for _, test := range []struct {
		Input    e.EntityInfo
		Mock     *gomock.Call
		Expected error
	}{
		{
			e.EntityInfo{
				Id:      "Tony",
				Node:    "test",
				Status:  1,
				History: "",
			},
			mq.
				EXPECT().
				Exec().
				Return(nil).
				Times(1),
			nil,
		},
		{
			e.EntityInfo{
				Id:      "Steve",
				Node:    "test1",
				Status:  1,
				History: "",
			},
			mq.
				EXPECT().
				Exec().
				Return(errors.New("error creating entity")).
				Times(1),
			errors.New("error creating entity"),
		},
	} {
		if err := dao.CreateEntity(test.Input); err != nil && err.Error() != test.Expected.Error() {
			t.Errorf("Got err: %+v, expected: %+v", err, test.Expected)
		}
	}
}

func TestClusterDaoImplCassandra_GetApps(t *testing.T) {
	dao, m, mq, mItr, ctrl := setupClusterDaoMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Consistency(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().PageSize(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Iter().Return(mItr).AnyTimes()

	for _, test := range []struct {
		Input    string
		Mock     []*gomock.Call
		Expected int
	}{
		{
			"",
			[]*gomock.Call{
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Tony").
					Return(true).
					Times(1),
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Steve").
					Return(true).
					Times(1),
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Thor").
					Return(false).
					Times(1),
				mItr.
					EXPECT().
					Close().
					Return(nil).
					Times(1),
			},
			2,
		},
		{
			"",
			[]*gomock.Call{
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "SRK").
					Return(true).
					Times(2),
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Ranbir").
					Return(true).
					Times(1),
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Alia").
					Return(false).
					Times(1),
				mItr.
					EXPECT().
					Close().
					Return(nil).
					Times(1),
			},
			3,
		},
		{
			"",
			[]*gomock.Call{
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "No One").
					Return(false).
					Times(1),
				mItr.
					EXPECT().
					Close().
					Return(nil).
					Times(1),
			},
			0,
		},
		{
			"Tom",
			[]*gomock.Call{
				mq.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Tom").
					Return(nil).
					Times(1),
			},
			1,
		},
	} {
		gomock.InOrder(test.Mock...)
		if apps, _ := dao.GetApps(test.Input); len(apps) != test.Expected {
			t.Errorf("Got length: %d, expected: %d", len(apps), test.Expected)
		}
	}

	for _, test := range []struct {
		Input    string
		Mock     []*gomock.Call
		Expected error
	}{
		{
			"",
			[]*gomock.Call{
				mItr.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "No One").
					Return(false).
					Times(1),
				mItr.
					EXPECT().
					Close().
					Return(errors.New("error closing iter")).
					AnyTimes(),
			},
			errors.New("error closing iter"),
		},
		{
			"Tom",
			[]*gomock.Call{
				mq.
					EXPECT().
					Scan(gomock.All()).
					SetArg(0, "Tom").
					Return(errors.New("error getting app")).
					AnyTimes(),
			},
			errors.New("error getting app"),
		},
	} {
		if _, err := dao.GetApps(test.Input); err.Error() != test.Expected.Error() {
			t.Errorf("Got error: %+v, expected: %+v", err, test.Expected)
		}
	}
}

func TestClusterDaoImplCassandra_UpdateAppActiveStatus(t *testing.T) {
	dao, m, mq, _, ctrl := setupClusterDaoMocks(t)
	defer ctrl.Finish()

	m.EXPECT().Query(gomock.Any()).Return(mq).AnyTimes()
	mq.EXPECT().Consistency(gomock.Any()).Return(mq).AnyTimes()

	for _, test := range []struct {
		Input    []interface{}
		Mock     *gomock.Call
		Expected error
	}{
		{
			[]interface{}{
				"Tony",
				true,
			},
			mq.
				EXPECT().
				Exec().
				Return(nil).
				Times(1),
			nil,
		},
		{
			[]interface{}{
				"Steve",
				false,
			},
			mq.
				EXPECT().
				Exec().
				Return(nil).
				Times(1),
			nil,
		},
		{
			[]interface{}{
				"Tom",
				true,
			},
			mq.
				EXPECT().
				Exec().
				Return(errors.New("error updating app Tom")).
				Times(1),
			errors.New("error updating app Tom"),
		},
	} {
		if err := dao.UpdateAppActiveStatus(test.Input[0].(string), test.Input[1].(bool)); err != nil && err.Error() != test.Expected.Error() {
			t.Errorf("Got err: %+v, expected: %+v", err, test.Expected)
		}
	}
}

func TestClusterDaoImplCassandra_InvalidateSingleAppCache(t *testing.T) {
	dao, _, _, _, ctrl := setupClusterDaoMocks(t)
	defer ctrl.Finish()

	dao.AppMap.m = map[string]store.App{
		"Tony": {
			AppId:      "Tony",
			Partitions: 5,
			Active:     true,
		},
		"Steve": {
			AppId:      "Steve",
			Partitions: 1,
			Active:     false,
		},
	}

	for _, test := range []struct {
		Input string
	}{
		{
			"Tony",
		},
		{
			"Steve",
		},
		{
			"Tom",
		},
	} {
		dao.InvalidateSingleAppCache(test.Input)
		if app, ok := dao.AppMap.m[test.Input]; ok {
			t.Errorf("Got app: %+v, expected no app", app)
		}
	}
}
