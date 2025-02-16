package services

import (
	"context"
	"net"
	"strconv"
	"testing"
	"time"

	"github.com/shellhub-io/shellhub/api/pkg/guard"
	"github.com/shellhub-io/shellhub/api/store"
	"github.com/shellhub-io/shellhub/api/store/mocks"
	"github.com/shellhub-io/shellhub/pkg/api/paginator"
	storecache "github.com/shellhub-io/shellhub/pkg/cache"
	"github.com/shellhub-io/shellhub/pkg/envs"
	"github.com/shellhub-io/shellhub/pkg/errors"
	"github.com/shellhub-io/shellhub/pkg/geoip"
	mocksGeoIp "github.com/shellhub-io/shellhub/pkg/geoip/mocks"
	"github.com/shellhub-io/shellhub/pkg/models"
	"github.com/shellhub-io/shellhub/pkg/validator"
	"github.com/stretchr/testify/assert"
)

func TestListDevices(t *testing.T) {
	mock := &mocks.Store{}
	s := NewService(store.Store(mock), privateKey, publicKey, storecache.NewNullCache(), clientMock, nil)

	ctx := context.TODO()

	Err := errors.New("error", "", 0)

	tenant := "tenant"

	devices := []models.Device{
		{UID: "uid"},
		{UID: "uid2"},
		{UID: "uid3"},
	}

	filters := []models.Filter{
		{
			Type:   "property",
			Params: &models.PropertyParams{Name: "hostname", Operator: "eq"},
		},
	}

	query := paginator.Query{Page: 1, PerPage: 10}

	status := []models.DeviceStatus{models.DeviceStatusPending, models.DeviceStatusAccepted, models.DeviceStatusRejected}
	sort := "name"
	order := []string{"asc", "desc"}

	namespace := &models.Namespace{
		TenantID:     tenant,
		MaxDevices:   3,
		DevicesCount: 3,
	}

	type Expected struct {
		devices []models.Device
		count   int
		err     error
	}

	cases := []struct {
		name          string
		tenant        string
		pagination    paginator.Query
		filter        []models.Filter
		status        models.DeviceStatus
		sort, order   string
		requiredMocks func()
		expected      Expected
	}{
		{
			name:       "ListDevices fails when the store device list fails when status is pending",
			tenant:     tenant,
			pagination: query,
			filter:     filters,
			status:     status[0],
			sort:       sort,
			order:      order[0],
			requiredMocks: func() {
				mock.On("NamespaceGet", ctx, namespace.TenantID).
					Return(namespace, nil).Once()
				mock.On("DeviceRemovedCount", ctx, namespace.TenantID).
					Return(int64(1), nil).Once()
				mock.On("DeviceList", ctx, query, filters, status[0], sort, order[0], store.DeviceListModeMaxDeviceReached).
					Return(nil, 0, Err).Once()
			},
			expected: Expected{
				nil,
				0,
				Err,
			},
		},
		{
			name:       "ListDevices fails when the store device list fails when status is not pending",
			tenant:     tenant,
			pagination: query,
			filter:     filters,
			status:     status[1],
			sort:       sort,
			order:      order[1],
			requiredMocks: func() {
				mock.On("DeviceList", ctx, query, filters, status[1], sort, order[1], store.DeviceListModeDefault).
					Return(nil, 0, Err).Once()
			},
			expected: Expected{
				nil,
				0,
				Err,
			},
		},
		{
			name:       "ListDevices succeeds when status is pending",
			tenant:     tenant,
			pagination: query,
			filter:     filters,
			status:     status[0],
			sort:       sort,
			order:      order[0],
			requiredMocks: func() {
				mock.On("NamespaceGet", ctx, namespace.TenantID).
					Return(namespace, nil).Once()
				mock.On("DeviceRemovedCount", ctx, namespace.TenantID).
					Return(int64(1), nil).Once()
				mock.On("DeviceList", ctx, query, filters, status[0], sort, order[0], store.DeviceListModeMaxDeviceReached).
					Return(devices, len(devices), nil).Once()
			},
			expected: Expected{
				devices,
				len(devices),
				nil,
			},
		},
		{
			name:       "ListDevices succeeds when status is not pending",
			tenant:     tenant,
			pagination: query,
			filter:     filters,
			status:     status[1],
			sort:       sort,
			order:      order[1],
			requiredMocks: func() {
				mock.On("DeviceList", ctx, query, filters, status[1], sort, order[1], store.DeviceListModeDefault).
					Return(devices, len(devices), nil).Once()
			},
			expected: Expected{
				devices,
				len(devices),
				nil,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(*testing.T) {
			tc.requiredMocks()
			returnedDevices, count, err := s.ListDevices(ctx, tc.tenant, tc.pagination, tc.filter, tc.status, tc.sort, tc.order)
			assert.Equal(t, tc.expected, Expected{returnedDevices, count, err})
		})
	}

	mock.AssertExpectations(t)
}

func TestGetDevice(t *testing.T) {
	mock := &mocks.Store{}
	s := NewService(store.Store(mock), privateKey, publicKey, storecache.NewNullCache(), clientMock, nil)

	Err := errors.New("error", "", 0)

	ctx := context.TODO()

	device := &models.Device{UID: "uid"}

	type Expected struct {
		device *models.Device
		err    error
	}

	cases := []struct {
		name          string
		requiredMocks func()
		uid           models.UID
		expected      Expected
	}{
		{
			name: "GetDevice fails when the store get device fails",
			requiredMocks: func() {
				mock.On("DeviceGet", ctx, models.UID("_uid")).
					Return(nil, Err).Once()
			},
			uid: models.UID("_uid"),
			expected: Expected{
				nil,
				NewErrDeviceNotFound(models.UID("_uid"), Err),
			},
		},
		{
			name: "GetDevice succeeds",
			requiredMocks: func() {
				mock.On("DeviceGet", ctx, models.UID(device.UID)).
					Return(device, nil).Once()
			},
			uid: models.UID("uid"),
			expected: Expected{
				device,
				nil,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.requiredMocks()
			returnedDevice, err := s.GetDevice(ctx, tc.uid)
			assert.Equal(t, tc.expected, Expected{returnedDevice, err})
		})
	}

	mock.AssertExpectations(t)
}

func TestDeleteDevice(t *testing.T) {
	mock := &mocks.Store{}
	s := NewService(store.Store(mock), privateKey, publicKey, storecache.NewNullCache(), clientMock, nil)

	envs.DefaultBackend = envMock

	ctx := context.TODO()

	user := &models.User{UserData: models.UserData{Name: "name", Email: "", Username: "username"}, ID: "id"}
	namespace := &models.Namespace{Name: "group1", Owner: "id", TenantID: "tenant", Members: []models.Member{{ID: "id", Role: guard.RoleOwner}, {ID: "id2", Role: guard.RoleObserver}}, MaxDevices: 3}
	device := &models.Device{UID: "uid", TenantID: "tenant", CreatedAt: time.Time{}}

	Err := errors.New("error", "", 0)

	cases := []struct {
		name          string
		requiredMocks func()
		uid           models.UID
		tenant, id    string
		expected      error
	}{
		{
			name:   "DeleteDevice fails when the store device get by uid fails",
			uid:    models.UID("_uid"),
			tenant: namespace.TenantID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID("_uid"), namespace.TenantID).
					Return(nil, Err).Once()
			},
			id:       user.ID,
			expected: NewErrDeviceNotFound(models.UID("_uid"), Err),
		},
		{
			name:   "DeleteDevice fails when the store namespace get fails",
			uid:    models.UID(device.UID),
			tenant: namespace.TenantID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).
					Return(device, nil).Once()
				mock.On("NamespaceGet", ctx, namespace.TenantID).
					Return(nil, Err).Once()
			},
			id:       user.ID,
			expected: NewErrNamespaceNotFound(namespace.TenantID, Err),
		},
		{
			name:   "DeleteDevice fails when device removed insert return error",
			uid:    models.UID(device.UID),
			tenant: namespace.TenantID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).
					Return(device, nil).Once()
				mock.On("NamespaceGet", ctx, namespace.TenantID).
					Return(namespace, nil).Once()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("true").Once()
				mock.On("DeviceRemovedInsert", ctx, namespace.TenantID, models.UID(device.UID)).
					Return(Err).Once()
			},
			id:       user.ID,
			expected: NewErrDeviceRemovedInsert(Err),
		},
		{
			name:   "DeleteDevice fails when the store device delete fails",
			uid:    models.UID(device.UID),
			tenant: namespace.TenantID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).
					Return(device, nil).Once()
				mock.On("NamespaceGet", ctx, namespace.TenantID).
					Return(namespace, nil).Once()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("false").Once()
				mock.On("DeviceDelete", ctx, models.UID(device.UID)).
					Return(Err).Once()
			},
			id:       user.ID,
			expected: Err,
		},
		{
			name:   "DeleteDevice succeeds",
			uid:    models.UID(device.UID),
			tenant: namespace.TenantID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).
					Return(device, nil).Once()
				mock.On("NamespaceGet", ctx, namespace.TenantID).
					Return(&models.Namespace{TenantID: namespace.TenantID}, nil).Once()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("false").Once()
				mock.On("DeviceDelete", ctx, models.UID(device.UID)).
					Return(nil).Once()
			},
			id:       user.ID,
			expected: nil,
		},
		{
			name:   "DeleteDevice fails to report usage",
			uid:    models.UID(device.UID),
			tenant: namespace.TenantID,
			requiredMocks: func() {
				namespaceBilling := &models.Namespace{
					Name:       "namespace1",
					MaxDevices: -1,
					Billing: &models.Billing{
						Active: true,
					},
				}
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).
					Return(device, nil).Once()
				mock.On("NamespaceGet", ctx, namespace.TenantID).
					Return(namespaceBilling, nil).Once()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("false").Once()
				clockMock.On("Now").Return(now).Twice()
				envMock.On("Get", "SHELLHUB_BILLING").Return(strconv.FormatBool(true)).Once()
				clientMock.On("ReportUsage", &models.UsageRecord{
					Device:    device,
					Namespace: namespaceBilling,
					Timestamp: now.Unix(),
				}).Return(500, nil).Once()
			},
			id:       user.ID,
			expected: ErrReport,
		},
		{
			name:   "DeleteDevice reports usage with success",
			uid:    models.UID(device.UID),
			tenant: namespace.TenantID,
			requiredMocks: func() {
				namespaceBilling := &models.Namespace{
					Name:    "namespace1",
					Members: []models.Member{{ID: "id", Role: guard.RoleOwner}, {ID: "id2", Role: guard.RoleObserver}},
					Billing: &models.Billing{
						Active: true,
					},
				}
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).
					Return(device, nil).Once()
				mock.On("NamespaceGet", ctx, namespace.TenantID).
					Return(namespaceBilling, nil).Once()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("false").Once()
				clockMock.On("Now").Return(now).Twice()
				envMock.On("Get", "SHELLHUB_BILLING").Return(strconv.FormatBool(true)).Once()
				clientMock.On("ReportUsage", &models.UsageRecord{
					Device:    device,
					Namespace: namespaceBilling,
					Timestamp: now.Unix(),
				}).Return(200, nil).Once()
				mock.On("DeviceDelete", ctx, models.UID(device.UID)).
					Return(nil).Once()
			},
			id:       user.ID,
			expected: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.requiredMocks()
			err := s.DeleteDevice(ctx, tc.uid, tc.tenant)
			assert.Equal(t, tc.expected, err)
		})
	}

	mock.AssertExpectations(t)
}

func TestRenameDevice(t *testing.T) {
	mock := &mocks.Store{}
	s := NewService(store.Store(mock), privateKey, publicKey, storecache.NewNullCache(), clientMock, nil)

	ctx := context.TODO()

	user := &models.User{UserData: models.UserData{Name: "name", Email: "email", Username: "username"}, ID: "id"}
	namespace := &models.Namespace{Name: "group1", Owner: "id", TenantID: "tenant", Members: []models.Member{{ID: "id", Role: guard.RoleOwner}, {ID: "id2", Role: guard.RoleObserver}}}
	device := &models.Device{UID: "uid", Name: "name", TenantID: "tenant", Identity: &models.DeviceIdentity{MAC: "00:00:00:00:00:00"}, Status: "accepted"}
	device2 := &models.Device{UID: "uid2", Name: "newname", TenantID: "tenant2"}
	Err := errors.New("error", "", 0)

	cases := []struct {
		name          string
		requiredMocks func()
		uid           models.UID
		expected      error
		deviceNewName string
		tenant, id    string
	}{
		{
			name:   "RenameDevice fails when store device get fails",
			tenant: namespace.TenantID,
			uid:    models.UID(device.UID),
			id:     user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).
					Return(device, Err).Once()
			},
			expected: NewErrDeviceNotFound(models.UID(device.UID), Err),
		},
		{
			name:          "RenameDevice fails when the name is invalid",
			tenant:        namespace.TenantID,
			deviceNewName: "---invalid...",
			uid:           models.UID(device.UID),
			id:            user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).
					Return(device, nil).Once()
			},
			expected: NewErrDeviceInvalid(map[string]interface{}{"Name": "---invalid..."}, validator.ErrInvalidFields),
		},
		{
			name:          "RenameDevice returns nil if the name is the same",
			tenant:        namespace.TenantID,
			deviceNewName: "name",
			uid:           models.UID(device.UID),
			id:            user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).
					Return(device, nil).Once()
			},
			expected: nil,
		},
		{
			name:          "RenameDevice fails when store get by device name fails",
			tenant:        namespace.TenantID,
			deviceNewName: "newname",
			uid:           models.UID(device.UID),
			id:            user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).Return(device, nil).Once()
				mock.On("DeviceGetByName", ctx, "newname", namespace.TenantID).
					Return(device2, Err).Once()
			},
			expected: NewErrDeviceNotFound(models.UID(device.UID), Err),
		},
		{
			name:          "RenameDevice fails when the name already exists",
			tenant:        namespace.TenantID,
			deviceNewName: "newname",
			uid:           models.UID(device.UID),
			id:            user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).Return(device, nil).Once()
				mock.On("DeviceGetByName", ctx, "newname", namespace.TenantID).
					Return(device2, nil).Once()
			},
			expected: NewErrDeviceDuplicated("newname", nil),
		},
		{
			name:          "RenameDevice fails when the store device rename fails",
			tenant:        namespace.TenantID,
			deviceNewName: "anewname",
			uid:           models.UID(device.UID),
			id:            user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).Return(device, nil).Once()
				mock.On("DeviceGetByName", ctx, "anewname", namespace.TenantID).
					Return(nil, store.ErrNoDocuments).Once()
				mock.On("DeviceRename", ctx, models.UID(device.UID), "anewname").
					Return(Err).Once()
			},
			expected: Err,
		},
		{
			name:          "RenameDevice succeeds",
			tenant:        namespace.TenantID,
			deviceNewName: "anewname",
			uid:           models.UID(device.UID),
			id:            user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).Return(device, nil).Once()
				mock.On("DeviceGetByName", ctx, "anewname", namespace.TenantID).
					Return(nil, store.ErrNoDocuments).Once()
				mock.On("DeviceRename", ctx, models.UID(device.UID), "anewname").
					Return(nil).Once()
			},
			expected: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.requiredMocks()
			err := s.RenameDevice(ctx, tc.uid, tc.deviceNewName, tc.tenant)
			assert.Equal(t, tc.expected, err)
		})
	}

	mock.AssertExpectations(t)
}

func TestLookupDevice(t *testing.T) {
	mock := &mocks.Store{}
	s := NewService(store.Store(mock), privateKey, publicKey, storecache.NewNullCache(), clientMock, nil)

	ctx := context.TODO()

	device := &models.Device{UID: "uid", Name: "name", TenantID: "tenant"}
	namespace := &models.Namespace{Name: "namespace"}
	Err := errors.New("error", "", 0)

	type Expected struct {
		device *models.Device
		err    error
	}

	cases := []struct {
		name          string
		namespace     string
		deviceName    string
		requiredMocks func()
		expected      Expected
	}{
		{
			name:       "LookupDevice fails when store device lookup fails",
			namespace:  namespace.Name,
			deviceName: device.Name,
			requiredMocks: func() {
				mock.On("DeviceLookup", ctx, namespace.Name, device.Name).
					Return(nil, Err).Once()
			},
			expected: Expected{
				nil,
				NewErrDeviceLookupNotFound(namespace.Name, device.Name, Err),
			},
		},
		{
			name:       "LookupDevice fails when the device is not found",
			namespace:  namespace.Name,
			deviceName: device.Name,
			requiredMocks: func() {
				mock.On("DeviceLookup", ctx, namespace.Name, device.Name).
					Return(nil, store.ErrNoDocuments).Once()
			},
			expected: Expected{
				nil,
				NewErrDeviceLookupNotFound(namespace.Name, device.Name, store.ErrNoDocuments),
			},
		},
		{
			name:       "LookupDevice succeeds",
			namespace:  namespace.Name,
			deviceName: device.Name,
			requiredMocks: func() {
				mock.On("DeviceLookup", ctx, namespace.Name, device.Name).
					Return(device, nil).Once()
			},
			expected: Expected{
				device,
				nil,
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.requiredMocks()
			returnedDevice, err := s.LookupDevice(ctx, tc.namespace, tc.deviceName)
			assert.Equal(t, tc.expected, Expected{returnedDevice, err})
		})
	}
	mock.AssertExpectations(t)
}

func TestUpdateDeviceStatus(t *testing.T) {
	mock := &mocks.Store{}
	s := NewService(store.Store(mock), privateKey, publicKey, storecache.NewNullCache(), clientMock, nil)

	Err := errors.New("error", "", 0)

	ctx := context.TODO()

	cases := []struct {
		name          string
		uid           models.UID
		online        bool
		requiredMocks func()
		expected      error
	}{
		{
			name: "UpdateDeviceStatus fails when store device online fails",
			uid:  models.UID("uid"),
			requiredMocks: func() {
				mock.On("DeviceSetOnline", ctx, models.UID("uid"), false).
					Return(Err).Once()
			},
			expected: Err,
		},
		{
			name:   "UpdateDeviceStatus succeeds",
			uid:    models.UID("uid"),
			online: true,
			requiredMocks: func() {
				online := true
				mock.On("DeviceSetOnline", ctx, models.UID("uid"), online).
					Return(Err).Once()
			},
			expected: Err,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.requiredMocks()
			err := s.UpdateDeviceStatus(ctx, tc.uid, tc.online)
			assert.Equal(t, tc.expected, err)
		})
	}

	mock.AssertExpectations(t)
}

func TestUpdatePendingStatus(t *testing.T) {
	mock := &mocks.Store{}
	s := NewService(store.Store(mock), privateKey, publicKey, storecache.NewNullCache(), clientMock, nil)

	envs.DefaultBackend = envMock

	user := &models.User{UserData: models.UserData{Name: "name", Username: "username"}, ID: "id"}
	namespace := &models.Namespace{Name: "group1", Owner: "id", TenantID: "tenant", MaxDevices: -1, Members: []models.Member{{ID: "id", Role: guard.RoleOwner}, {ID: "id2", Role: guard.RoleObserver}}}
	namespaceWithLimit := &models.Namespace{Name: "group1", Owner: "id", TenantID: "tenant", MaxDevices: 3, DevicesCount: 1, Members: []models.Member{{ID: "id", Role: guard.RoleOwner}, {ID: "id2", Role: guard.RoleObserver}}}
	identity := &models.DeviceIdentity{MAC: "mac"}
	device := &models.Device{UID: "uid", Name: "name", TenantID: "tenant", Identity: identity, CreatedAt: time.Time{}}

	Err := errors.New("error", "", 0)

	ctx := context.TODO()

	cases := []struct {
		name          string
		uid           models.UID
		status        models.DeviceStatus
		tenant, id    string
		requiredMocks func()
		expected      error
	}{
		{
			name:   "UpdatePendingStatus fails when the status is invalid",
			uid:    models.UID("uid"),
			tenant: namespace.TenantID,
			status: models.DeviceStatus("invalid"),
			id:     user.ID,
			requiredMocks: func() {
			},
			expected: NewErrDeviceStatusInvalid("invalid", nil),
		},
		{
			name:   "UpdatePendingStatus fails when the store get by uid fails",
			uid:    models.UID("uid"),
			tenant: namespace.TenantID,
			status: models.DeviceStatusAccepted,
			id:     user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID("uid"), namespace.TenantID).
					Return(nil, Err).Once()
			},
			expected: NewErrDeviceNotFound("uid", Err),
		},
		{
			name:   "UpdatePendingStatus fails when device already accepted",
			uid:    models.UID("uid"),
			status: models.DeviceStatusAccepted,
			tenant: namespace.TenantID,
			id:     user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID("uid"), namespace.TenantID).
					Return(&models.Device{Status: "accepted"}, nil).Once()
			},
			expected: NewErrDeviceStatusAccepted(nil),
		},
		{
			name:   "UpdatePendingStatus fails to update accept",
			uid:    models.UID("uid"),
			status: models.DeviceStatusPending,
			tenant: namespace.TenantID,
			id:     user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID("uid"), namespace.TenantID).Return(&models.Device{Status: "accepted"}, nil).Once()
			},
			expected: NewErrDeviceStatusAccepted(nil),
		},
		{
			name:   "UpdatePendingStatus fail when could not get namespace",
			uid:    models.UID(device.UID),
			status: models.DeviceStatusAccepted,
			tenant: namespace.TenantID,
			id:     user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID("uid"), namespace.TenantID).
					Return(device, nil).Once()
				mock.On("NamespaceGet", ctx, namespace.TenantID).
					Return(nil, Err).Once()
			},
			expected: NewErrNamespaceNotFound(namespace.TenantID, Err),
		},
		{
			name:   "Test should fail when device removed get return a error that is not store.ErrNoDocuments",
			uid:    models.UID(device.UID),
			status: models.DeviceStatusAccepted,
			tenant: namespaceWithLimit.TenantID,
			id:     user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID("uid"), namespaceWithLimit.TenantID).
					Return(device, nil).Once()
				mock.On("NamespaceGet", ctx, namespaceWithLimit.TenantID).
					Return(namespaceWithLimit, nil).Once()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("true").Once()
				mock.On("DeviceRemovedGet", ctx, namespaceWithLimit.TenantID, models.UID(device.UID)).
					Return(nil, Err).Once()
			},
			expected: NewErrDeviceRemovedGet(Err),
		},
		{
			name:   "Test should fail when device is not removed, but device removed list return a error",
			uid:    models.UID(device.UID),
			status: models.DeviceStatusAccepted,
			tenant: namespaceWithLimit.TenantID,
			id:     user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID("uid"), namespaceWithLimit.TenantID).
					Return(device, nil).Once()
				mock.On("NamespaceGet", ctx, namespaceWithLimit.TenantID).
					Return(namespaceWithLimit, nil).Once()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("true").Once()
				mock.On("DeviceRemovedGet", ctx, namespaceWithLimit.TenantID, models.UID(device.UID)).
					Return(nil, store.ErrNoDocuments).Once()
				mock.On("DeviceRemovedCount", ctx, namespaceWithLimit.TenantID).
					Return(int64(0), Err).Once()
			},
			expected: NewErrDeviceRemovedCount(Err),
		},
		{
			name:   "Test should fail when device is not removed, but the device limit has been reached",
			uid:    models.UID(device.UID),
			status: models.DeviceStatusAccepted,
			tenant: namespaceWithLimit.TenantID,
			id:     user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID("uid"), namespaceWithLimit.TenantID).
					Return(device, nil).Once()
				mock.On("NamespaceGet", ctx, namespaceWithLimit.TenantID).
					Return(namespaceWithLimit, nil).Once()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("true").Once()
				mock.On("DeviceRemovedGet", ctx, namespaceWithLimit.TenantID, models.UID(device.UID)).
					Return(nil, store.ErrNoDocuments).Once()
				mock.On("DeviceRemovedCount", ctx, namespaceWithLimit.TenantID).
					Return(int64(2), nil).Once()
			},
			expected: NewErrDeviceRemovedFull(namespaceWithLimit.MaxDevices, nil),
		},
		{
			name:   "Test should fail when device was removed, but device removed delete return a error",
			uid:    models.UID(device.UID),
			status: models.DeviceStatusAccepted,
			tenant: namespaceWithLimit.TenantID,
			id:     user.ID,
			requiredMocks: func() {
				mock.On("DeviceGetByUID", ctx, models.UID("uid"), namespaceWithLimit.TenantID).
					Return(device, nil).Once()
				mock.On("NamespaceGet", ctx, namespaceWithLimit.TenantID).
					Return(namespaceWithLimit, nil).Once()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("true").Once()
				mock.On("DeviceRemovedGet", ctx, namespaceWithLimit.TenantID, models.UID(device.UID)).
					Return(&models.DeviceRemoved{
						UID:    models.UID(device.UID),
						Tenant: namespaceWithLimit.TenantID,
					}, nil).Once()
				mock.On("DeviceRemovedDelete", ctx, namespaceWithLimit.TenantID, models.UID(device.UID)).
					Return(Err).Once()
			},
			expected: NewErrDeviceRemovedDelete(Err),
		},
		{
			name:   "UpdatePendingStatus fails when the limit is exceeded",
			uid:    models.UID("uid_limit"),
			status: models.DeviceStatusAccepted,
			tenant: "tenant_max",
			id:     user.ID,
			requiredMocks: func() {
				namespaceExceedLimit := &models.Namespace{Name: "group1", Owner: "id", TenantID: "tenant_max", MaxDevices: 3, DevicesCount: 3, Members: []models.Member{{ID: "id", Role: guard.RoleOwner}, {ID: "id2", Role: guard.RoleObserver}}}
				deviceExceed := &models.Device{UID: "uid_limit", Name: "name", TenantID: "tenant_max", Identity: identity, Status: "pending"}
				mock.On("DeviceGetByUID", ctx, models.UID(deviceExceed.UID), deviceExceed.TenantID).
					Return(deviceExceed, nil).Once()
				mock.On("NamespaceGet", ctx, deviceExceed.TenantID).
					Return(namespaceExceedLimit, nil).Twice()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("false").Once()
				mock.On("DeviceGetByMac", ctx, "mac", deviceExceed.TenantID, models.DeviceStatusAccepted).
					Return(nil, nil).Once()
			},
			expected: NewErrDeviceLimit(3, nil),
		},
		{
			name:   "UpdatePendingStatus succeeds",
			uid:    models.UID("uid"),
			status: models.DeviceStatusAccepted,
			tenant: namespace.TenantID,
			id:     user.ID,
			requiredMocks: func() {
				oldDevice := &models.Device{UID: "uid2", Name: "name", TenantID: "tenant", Identity: identity}
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), namespace.TenantID).
					Return(device, nil).Once()
				mock.On("NamespaceGet", ctx, namespace.TenantID).
					Return(namespace, nil).Once()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("false").Once()
				mock.On("DeviceGetByMac", ctx, "mac", device.TenantID, models.DeviceStatusAccepted).
					Return(oldDevice, nil).Once()
				mock.On("SessionUpdateDeviceUID", ctx, models.UID(oldDevice.UID), models.UID(device.UID)).
					Return(nil).Once()
				mock.On("DeviceDelete", ctx, models.UID(oldDevice.UID)).
					Return(nil).Once()
				mock.On("DeviceRename", ctx, models.UID(device.UID), oldDevice.Name).
					Return(nil).Once()
				mock.On("DeviceUpdateStatus", ctx, models.UID(device.UID), models.DeviceStatusAccepted).
					Return(nil).Once()
			},
			expected: nil,
		},
		{
			name:   "UpdatePendingStatus reports usage",
			uid:    models.UID("uid"),
			status: models.DeviceStatusAccepted,
			tenant: "tenant_max",
			id:     user.ID,
			requiredMocks: func() {
				namespaceBilling := &models.Namespace{Name: "group1", Owner: "id", TenantID: "tenant_max", MaxDevices: -1, DevicesCount: 10, Billing: &models.Billing{Active: true}, Members: []models.Member{{ID: "id", Role: guard.RoleOwner}, {ID: "id2", Role: guard.RoleObserver}}}
				device := &models.Device{UID: "uid", Name: "name", TenantID: "tenant_max", Identity: identity, Status: "pending"}
				mock.On("NamespaceGet", ctx, device.TenantID).
					Return(namespaceBilling, nil).Once()
				mock.On("NamespaceGet", ctx, namespaceBilling.TenantID).
					Return(namespaceBilling, nil).Once()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("false").Once()
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), device.TenantID).
					Return(device, nil).Once()
				mock.On("DeviceGetByMac", ctx, "mac", device.TenantID, models.DeviceStatusAccepted).
					Return(nil, nil).Once()
				clockMock.On("Now").Return(now).Twice()
				envMock.On("Get", "SHELLHUB_BILLING").Return(strconv.FormatBool(true)).Once()
				clientMock.On("ReportUsage", &models.UsageRecord{
					Device:    device,
					Inc:       true,
					Namespace: namespaceBilling,
					Timestamp: now.Unix(),
				}).Return(200, nil).Once()
				mock.On("DeviceUpdateStatus", ctx, models.UID(device.UID), models.DeviceStatusAccepted).
					Return(nil).Once()
			},
			expected: nil,
		},
		{
			name:   "UpdatePendingStatus fails to reports usage",
			uid:    models.UID("uid"),
			status: models.DeviceStatusAccepted,
			tenant: "tenant_max",
			id:     user.ID,
			requiredMocks: func() {
				namespaceBilling := &models.Namespace{Name: "group1", Owner: "id", TenantID: "tenant_max", MaxDevices: -1, DevicesCount: 10, Billing: &models.Billing{Active: true}, Members: []models.Member{{ID: "id", Role: guard.RoleOwner}, {ID: "id2", Role: guard.RoleObserver}}}
				device := &models.Device{UID: "uid", Name: "name", TenantID: "tenant_max", Identity: identity, Status: "pending"}
				mock.On("NamespaceGet", ctx, device.TenantID).
					Return(namespaceBilling, nil).Once()
				mock.On("NamespaceGet", ctx, namespaceBilling.TenantID).
					Return(namespaceBilling, nil).Once()
				envMock.On("Get", "SHELLHUB_CLOUD").Return("false").Once()
				mock.On("DeviceGetByUID", ctx, models.UID(device.UID), device.TenantID).
					Return(device, nil).Once()
				mock.On("DeviceGetByMac", ctx, "mac", device.TenantID, models.DeviceStatusAccepted).
					Return(nil, nil).Once()
				clockMock.On("Now").Return(now).Twice()
				envMock.On("Get", "SHELLHUB_BILLING").Return(strconv.FormatBool(true)).Once()
				clientMock.On("ReportUsage", &models.UsageRecord{
					Namespace: namespaceBilling,
					Inc:       true,
					Device:    device,
					Timestamp: now.Unix(),
				}).Return(500, nil).Once()
			},
			expected: ErrReport,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.requiredMocks()
			err := s.UpdatePendingStatus(ctx, tc.uid, tc.status, tc.tenant)
			assert.Equal(t, tc.expected, err)
		})
	}

	mock.AssertExpectations(t)
}

func TestSetDevicePosition(t *testing.T) {
	locator := &mocksGeoIp.Locator{}
	mock := &mocks.Store{}
	s := NewService(store.Store(mock), privateKey, publicKey, storecache.NewNullCache(), clientMock, locator)

	ctx := context.TODO()

	device := &models.Device{UID: "uid"}

	Err := errors.New("error", "", 0)

	positionGeoIP := geoip.Position{Longitude: 0, Latitude: 0}
	positionDeviceModel := models.DevicePosition{Longitude: 0, Latitude: 0}

	cases := []struct {
		name          string
		requiredMocks func()
		uid           models.UID
		ip            string
		expected      error
	}{
		{
			name: "SetDevicePosition fails when DeviceSetPosition return error",
			requiredMocks: func() {
				locator.On("GetPosition", net.ParseIP("127.0.0.1")).
					Return(positionGeoIP, nil).Once()
				mock.On("DeviceSetPosition", ctx, models.UID(device.UID), positionDeviceModel).
					Return(Err).Once()
			},
			uid:      models.UID(device.UID),
			ip:       "127.0.0.1",
			expected: Err,
		},
		{
			name: "SetDevicePosition success",
			requiredMocks: func() {
				locator.On("GetPosition", net.ParseIP("127.0.0.1")).
					Return(positionGeoIP, nil).Once()
				mock.On("DeviceSetPosition", ctx, models.UID(device.UID), positionDeviceModel).
					Return(nil).Once()
			},
			uid:      models.UID(device.UID),
			ip:       "127.0.0.1",
			expected: nil,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			tc.requiredMocks()

			err := s.SetDevicePosition(ctx, tc.uid, tc.ip)
			assert.Equal(t, tc.expected, err)
		})
	}

	mock.AssertExpectations(t)
}

func TestDeviceHeartbeat(t *testing.T) {
	mock := &mocks.Store{}
	s := NewService(store.Store(mock), privateKey, publicKey, storecache.NewNullCache(), clientMock, nil)

	ctx := context.TODO()
	uid := models.UID("uid")

	clockMock.On("Now").Return(now).Once()

	mock.On("DeviceSetOnline", ctx, uid, true).Return(nil).Once()

	err := s.DeviceHeartbeat(ctx, uid)
	assert.NoError(t, err)

	mock.AssertExpectations(t)
}
