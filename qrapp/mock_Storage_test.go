// Code generated by mockery. DO NOT EDIT.

package qrapp

import (
	context "context"
	fs "io/fs"

	io "io"

	mock "github.com/stretchr/testify/mock"
)

// MockStorage is an autogenerated mock type for the Storage type
type MockStorage struct {
	mock.Mock
}

// Delete provides a mock function with given fields: ctx, bucket, key
func (_m *MockStorage) Delete(ctx context.Context, bucket string, key string) error {
	ret := _m.Called(ctx, bucket, key)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string) error); ok {
		r0 = rf(ctx, bucket, key)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// DownloadToTmpFile provides a mock function with given fields: ctx, bucket, key
func (_m *MockStorage) DownloadToTmpFile(ctx context.Context, bucket string, key string) (fs.File, error) {
	ret := _m.Called(ctx, bucket, key)

	var r0 fs.File
	if rf, ok := ret.Get(0).(func(context.Context, string, string) fs.File); ok {
		r0 = rf(ctx, bucket, key)
	} else {
		if ret.Get(0) != nil {
			r0 = ret.Get(0).(fs.File)
		}
	}

	var r1 error
	if rf, ok := ret.Get(1).(func(context.Context, string, string) error); ok {
		r1 = rf(ctx, bucket, key)
	} else {
		r1 = ret.Error(1)
	}

	return r0, r1
}

// RemoveTmpFile provides a mock function with given fields: ctx, tmpFile
func (_m *MockStorage) RemoveTmpFile(ctx context.Context, tmpFile fs.File) error {
	ret := _m.Called(ctx, tmpFile)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, fs.File) error); ok {
		r0 = rf(ctx, tmpFile)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}

// Upload provides a mock function with given fields: ctx, bucket, key, contentType, r
func (_m *MockStorage) Upload(ctx context.Context, bucket string, key string, contentType string, r io.Reader) error {
	ret := _m.Called(ctx, bucket, key, contentType, r)

	var r0 error
	if rf, ok := ret.Get(0).(func(context.Context, string, string, string, io.Reader) error); ok {
		r0 = rf(ctx, bucket, key, contentType, r)
	} else {
		r0 = ret.Error(0)
	}

	return r0
}
