package query

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"regexp"
	"testing"

	sq "github.com/Masterminds/squirrel"
	"golang.org/x/text/language"

	"github.com/zitadel/zitadel/internal/database"
	"github.com/zitadel/zitadel/internal/domain"
	errs "github.com/zitadel/zitadel/internal/errors"
)

var (
	loginNamesQuery = `SELECT login_names.user_id, ARRAY_AGG(login_names.login_name)::TEXT[] AS loginnames, ARRAY_AGG(LOWER(login_names.login_name))::TEXT[] AS loginnames_lower, login_names.instance_id, login_names.user_owner_removed, login_names.policy_owner_removed, login_names.domain_owner_removed` +
		` FROM projections.login_names2 AS login_names` +
		` GROUP BY login_names.user_id, login_names.instance_id, login_names.user_owner_removed, login_names.policy_owner_removed, login_names.domain_owner_removed`
	preferredLoginNameQuery = `SELECT preferred_login_name.user_id, preferred_login_name.login_name, preferred_login_name.instance_id, preferred_login_name.user_owner_removed, preferred_login_name.policy_owner_removed, preferred_login_name.domain_owner_removed` +
		` FROM projections.login_names2 AS preferred_login_name` +
		` WHERE  preferred_login_name.is_primary = $1`
	userQuery = `SELECT projections.users6.id,` +
		` projections.users6.creation_date,` +
		` projections.users6.change_date,` +
		` projections.users6.resource_owner,` +
		` projections.users6.sequence,` +
		` projections.users6.state,` +
		` projections.users6.type,` +
		` projections.users6.username,` +
		` login_names.loginnames,` +
		` preferred_login_name.login_name,` +
		` projections.users6_humans.user_id,` +
		` projections.users6_humans.first_name,` +
		` projections.users6_humans.last_name,` +
		` projections.users6_humans.nick_name,` +
		` projections.users6_humans.display_name,` +
		` projections.users6_humans.preferred_language,` +
		` projections.users6_humans.gender,` +
		` projections.users6_humans.avatar_key,` +
		` projections.users6_humans.email,` +
		` projections.users6_humans.is_email_verified,` +
		` projections.users6_humans.phone,` +
		` projections.users6_humans.is_phone_verified,` +
		` projections.users6_machines.user_id,` +
		` projections.users6_machines.name,` +
		` projections.users6_machines.description,` +
		` COUNT(*) OVER ()` +
		` FROM projections.users6` +
		` LEFT JOIN projections.users6_humans ON projections.users6.id = projections.users6_humans.user_id AND projections.users6.instance_id = projections.users6_humans.instance_id` +
		` LEFT JOIN projections.users6_machines ON projections.users6.id = projections.users6_machines.user_id AND projections.users6.instance_id = projections.users6_machines.instance_id` +
		` LEFT JOIN` +
		` (` + loginNamesQuery + `) AS login_names` +
		` ON login_names.user_id = projections.users6.id AND login_names.instance_id = projections.users6.instance_id` +
		` LEFT JOIN` +
		` (` + preferredLoginNameQuery + `) AS preferred_login_name` +
		` ON preferred_login_name.user_id = projections.users6.id AND preferred_login_name.instance_id = projections.users6.instance_id`
	userCols = []string{
		"id",
		"creation_date",
		"change_date",
		"resource_owner",
		"sequence",
		"state",
		"type",
		"username",
		"loginnames",
		"login_name",
		//human
		"user_id",
		"first_name",
		"last_name",
		"nick_name",
		"display_name",
		"preferred_language",
		"gender",
		"avatar_key",
		"email",
		"is_email_verified",
		"phone",
		"is_phone_verified",
		//machine
		"user_id",
		"name",
		"description",
		"count",
	}
	profileQuery = `SELECT projections.users6.id,` +
		` projections.users6.creation_date,` +
		` projections.users6.change_date,` +
		` projections.users6.resource_owner,` +
		` projections.users6.sequence,` +
		` projections.users6_humans.user_id,` +
		` projections.users6_humans.first_name,` +
		` projections.users6_humans.last_name,` +
		` projections.users6_humans.nick_name,` +
		` projections.users6_humans.display_name,` +
		` projections.users6_humans.preferred_language,` +
		` projections.users6_humans.gender,` +
		` projections.users6_humans.avatar_key` +
		` FROM projections.users6` +
		` LEFT JOIN projections.users6_humans ON projections.users6.id = projections.users6_humans.user_id AND projections.users6.instance_id = projections.users6_humans.instance_id`
	profileCols = []string{
		"id",
		"creation_date",
		"change_date",
		"resource_owner",
		"sequence",
		"user_id",
		"first_name",
		"last_name",
		"nick_name",
		"display_name",
		"preferred_language",
		"gender",
		"avatar_key",
	}
	emailQuery = `SELECT projections.users6.id,` +
		` projections.users6.creation_date,` +
		` projections.users6.change_date,` +
		` projections.users6.resource_owner,` +
		` projections.users6.sequence,` +
		` projections.users6_humans.user_id,` +
		` projections.users6_humans.email,` +
		` projections.users6_humans.is_email_verified` +
		` FROM projections.users6` +
		` LEFT JOIN projections.users6_humans ON projections.users6.id = projections.users6_humans.user_id AND projections.users6.instance_id = projections.users6_humans.instance_id`
	emailCols = []string{
		"id",
		"creation_date",
		"change_date",
		"resource_owner",
		"sequence",
		"user_id",
		"email",
		"is_email_verified",
	}
	phoneQuery = `SELECT projections.users6.id,` +
		` projections.users6.creation_date,` +
		` projections.users6.change_date,` +
		` projections.users6.resource_owner,` +
		` projections.users6.sequence,` +
		` projections.users6_humans.user_id,` +
		` projections.users6_humans.phone,` +
		` projections.users6_humans.is_phone_verified` +
		` FROM projections.users6` +
		` LEFT JOIN projections.users6_humans ON projections.users6.id = projections.users6_humans.user_id AND projections.users6.instance_id = projections.users6_humans.instance_id`
	phoneCols = []string{
		"id",
		"creation_date",
		"change_date",
		"resource_owner",
		"sequence",
		"user_id",
		"phone",
		"is_phone_verified",
	}
	userUniqueQuery = `SELECT projections.users6.id,` +
		` projections.users6.state,` +
		` projections.users6.username,` +
		` projections.users6_humans.user_id,` +
		` projections.users6_humans.email,` +
		` projections.users6_humans.is_email_verified` +
		` FROM projections.users6` +
		` LEFT JOIN projections.users6_humans ON projections.users6.id = projections.users6_humans.user_id AND projections.users6.instance_id = projections.users6_humans.instance_id`
	userUniqueCols = []string{
		"id",
		"state",
		"username",
		"user_id",
		"email",
		"is_email_verified",
	}
	notifyUserQuery = `SELECT projections.users6.id,` +
		` projections.users6.creation_date,` +
		` projections.users6.change_date,` +
		` projections.users6.resource_owner,` +
		` projections.users6.sequence,` +
		` projections.users6.state,` +
		` projections.users6.type,` +
		` projections.users6.username,` +
		` login_names.loginnames,` +
		` preferred_login_name.login_name,` +
		` projections.users6_humans.user_id,` +
		` projections.users6_humans.first_name,` +
		` projections.users6_humans.last_name,` +
		` projections.users6_humans.nick_name,` +
		` projections.users6_humans.display_name,` +
		` projections.users6_humans.preferred_language,` +
		` projections.users6_humans.gender,` +
		` projections.users6_humans.avatar_key,` +
		` projections.users6_notifications.user_id,` +
		` projections.users6_notifications.last_email,` +
		` projections.users6_notifications.verified_email,` +
		` projections.users6_notifications.last_phone,` +
		` projections.users6_notifications.verified_phone,` +
		` projections.users6_notifications.password_set,` +
		` COUNT(*) OVER ()` +
		` FROM projections.users6` +
		` LEFT JOIN projections.users6_humans ON projections.users6.id = projections.users6_humans.user_id AND projections.users6.instance_id = projections.users6_humans.instance_id` +
		` LEFT JOIN projections.users6_notifications ON projections.users6.id = projections.users6_notifications.user_id AND projections.users6.instance_id = projections.users6_notifications.instance_id` +
		` LEFT JOIN` +
		` (` + loginNamesQuery + `) AS login_names` +
		` ON login_names.user_id = projections.users6.id AND login_names.instance_id = projections.users6.instance_id` +
		` LEFT JOIN` +
		` (` + preferredLoginNameQuery + `) AS preferred_login_name` +
		` ON preferred_login_name.user_id = projections.users6.id AND preferred_login_name.instance_id = projections.users6.instance_id`
	notifyUserCols = []string{
		"id",
		"creation_date",
		"change_date",
		"resource_owner",
		"sequence",
		"state",
		"type",
		"username",
		"loginnames",
		"login_name",
		//human
		"user_id",
		"first_name",
		"last_name",
		"nick_name",
		"display_name",
		"preferred_language",
		"gender",
		"avatar_key",
		//machine
		"user_id",
		"last_email",
		"verified_email",
		"last_phone",
		"verified_phone",
		"password_set",
		"count",
	}
	usersQuery = `SELECT projections.users6.id,` +
		` projections.users6.creation_date,` +
		` projections.users6.change_date,` +
		` projections.users6.resource_owner,` +
		` projections.users6.sequence,` +
		` projections.users6.state,` +
		` projections.users6.type,` +
		` projections.users6.username,` +
		` login_names.loginnames,` +
		` preferred_login_name.login_name,` +
		` projections.users6_humans.user_id,` +
		` projections.users6_humans.first_name,` +
		` projections.users6_humans.last_name,` +
		` projections.users6_humans.nick_name,` +
		` projections.users6_humans.display_name,` +
		` projections.users6_humans.preferred_language,` +
		` projections.users6_humans.gender,` +
		` projections.users6_humans.avatar_key,` +
		` projections.users6_humans.email,` +
		` projections.users6_humans.is_email_verified,` +
		` projections.users6_humans.phone,` +
		` projections.users6_humans.is_phone_verified,` +
		` projections.users6_machines.user_id,` +
		` projections.users6_machines.name,` +
		` projections.users6_machines.description,` +
		` COUNT(*) OVER ()` +
		` FROM projections.users6` +
		` LEFT JOIN projections.users6_humans ON projections.users6.id = projections.users6_humans.user_id AND projections.users6.instance_id = projections.users6_humans.instance_id` +
		` LEFT JOIN projections.users6_machines ON projections.users6.id = projections.users6_machines.user_id AND projections.users6.instance_id = projections.users6_machines.instance_id` +
		` LEFT JOIN` +
		` (` + loginNamesQuery + `) AS login_names` +
		` ON login_names.user_id = projections.users6.id AND login_names.instance_id = projections.users6.instance_id` +
		` LEFT JOIN` +
		` (` + preferredLoginNameQuery + `) AS preferred_login_name` +
		` ON preferred_login_name.user_id = projections.users6.id AND preferred_login_name.instance_id = projections.users6.instance_id`
	usersCols = []string{
		"id",
		"creation_date",
		"change_date",
		"resource_owner",
		"sequence",
		"state",
		"type",
		"username",
		"loginnames",
		"login_name",
		//human
		"user_id",
		"first_name",
		"last_name",
		"nick_name",
		"display_name",
		"preferred_language",
		"gender",
		"avatar_key",
		"email",
		"is_email_verified",
		"phone",
		"is_phone_verified",
		//machine
		"user_id",
		"name",
		"description",
		"count",
	}
)

func Test_UserPrepares(t *testing.T) {
	type want struct {
		sqlExpectations sqlExpectation
		err             checkErr
	}
	tests := []struct {
		name    string
		prepare interface{}
		want    want
		object  interface{}
	}{
		{
			name: "prepareUserQuery no result",
			prepare: func() (sq.SelectBuilder, func(*sql.Row) (*User, error)) {
				return prepareUserQuery()
			},
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(userQuery),
					nil,
					nil,
				),
				err: func(err error) (error, bool) {
					if !errs.IsNotFound(err) {
						return fmt.Errorf("err should be zitadel.NotFoundError got: %w", err), false
					}
					return nil, true
				},
			},
			object: (*User)(nil),
		},
		{
			name: "prepareUserQuery human found",
			prepare: func() (sq.SelectBuilder, func(*sql.Row) (*User, error)) {
				return prepareUserQuery()
			},
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(userQuery),
					userCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						"resource_owner",
						uint64(20211108),
						domain.UserStateActive,
						domain.UserTypeHuman,
						"username",
						database.StringArray{"login_name1", "login_name2"},
						"login_name1",
						//human
						"id",
						"first_name",
						"last_name",
						"nick_name",
						"display_name",
						"de",
						domain.GenderUnspecified,
						"avatar_key",
						"email",
						true,
						"phone",
						true,
						//machine
						nil,
						nil,
						nil,
						1,
					},
				),
			},
			object: &User{
				ID:                 "id",
				CreationDate:       testNow,
				ChangeDate:         testNow,
				ResourceOwner:      "resource_owner",
				Sequence:           20211108,
				State:              domain.UserStateActive,
				Type:               domain.UserTypeHuman,
				Username:           "username",
				LoginNames:         database.StringArray{"login_name1", "login_name2"},
				PreferredLoginName: "login_name1",
				Human: &Human{
					FirstName:         "first_name",
					LastName:          "last_name",
					NickName:          "nick_name",
					DisplayName:       "display_name",
					AvatarKey:         "avatar_key",
					PreferredLanguage: language.German,
					Gender:            domain.GenderUnspecified,
					Email:             "email",
					IsEmailVerified:   true,
					Phone:             "phone",
					IsPhoneVerified:   true,
				},
			},
		},
		{
			name: "prepareUserQuery machine found",
			prepare: func() (sq.SelectBuilder, func(*sql.Row) (*User, error)) {
				return prepareUserQuery()
			},
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(userQuery),
					userCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						"resource_owner",
						uint64(20211108),
						domain.UserStateActive,
						domain.UserTypeMachine,
						"username",
						database.StringArray{"login_name1", "login_name2"},
						"login_name1",
						//human
						nil,
						nil,
						nil,
						nil,
						nil,
						nil,
						nil,
						nil,
						nil,
						nil,
						nil,
						nil,
						//machine
						"id",
						"name",
						"description",
						1,
					},
				),
			},
			object: &User{
				ID:                 "id",
				CreationDate:       testNow,
				ChangeDate:         testNow,
				ResourceOwner:      "resource_owner",
				Sequence:           20211108,
				State:              domain.UserStateActive,
				Type:               domain.UserTypeMachine,
				Username:           "username",
				LoginNames:         database.StringArray{"login_name1", "login_name2"},
				PreferredLoginName: "login_name1",
				Machine: &Machine{
					Name:        "name",
					Description: "description",
				},
			},
		},
		{
			name: "prepareUserQuery sql err",
			prepare: func() (sq.SelectBuilder, func(*sql.Row) (*User, error)) {
				return prepareUserQuery()
			},
			want: want{
				sqlExpectations: mockQueryErr(
					regexp.QuoteMeta(userQuery),
					sql.ErrConnDone,
				),
				err: func(err error) (error, bool) {
					if !errors.Is(err, sql.ErrConnDone) {
						return fmt.Errorf("err should be sql.ErrConnDone got: %w", err), false
					}
					return nil, true
				},
			},
			object: nil,
		},
		{
			name:    "prepareProfileQuery no result",
			prepare: prepareProfileQuery,
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(profileQuery),
					nil,
					nil,
				),
				err: func(err error) (error, bool) {
					if !errs.IsNotFound(err) {
						return fmt.Errorf("err should be zitadel.NotFoundError got: %w", err), false
					}
					return nil, true
				},
			},
			object: (*Profile)(nil),
		},
		{
			name:    "prepareProfileQuery human found",
			prepare: prepareProfileQuery,
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(profileQuery),
					profileCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						"resource_owner",
						uint64(20211108),
						"id",
						"first_name",
						"last_name",
						"nick_name",
						"display_name",
						"de",
						domain.GenderUnspecified,
						"avatar_key",
					},
				),
			},
			object: &Profile{
				ID:                "id",
				CreationDate:      testNow,
				ChangeDate:        testNow,
				ResourceOwner:     "resource_owner",
				Sequence:          20211108,
				FirstName:         "first_name",
				LastName:          "last_name",
				NickName:          "nick_name",
				DisplayName:       "display_name",
				AvatarKey:         "avatar_key",
				PreferredLanguage: language.German,
				Gender:            domain.GenderUnspecified,
			},
		},
		{
			name:    "prepareProfileQuery not human found (error)",
			prepare: prepareProfileQuery,
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(profileQuery),
					profileCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						"resource_owner",
						uint64(20211108),
						nil,
						nil,
						nil,
						nil,
						nil,
						nil,
						nil,
						nil,
					},
				),
				err: func(err error) (error, bool) {
					if !errs.IsPreconditionFailed(err) {
						return fmt.Errorf("err should be zitadel.PredconditionError got: %w", err), false
					}
					return nil, true
				},
			},
			object: (*Profile)(nil),
		},
		{
			name:    "prepareProfileQuery sql err",
			prepare: prepareProfileQuery,
			want: want{
				sqlExpectations: mockQueryErr(
					regexp.QuoteMeta(profileQuery),
					sql.ErrConnDone,
				),
				err: func(err error) (error, bool) {
					if !errors.Is(err, sql.ErrConnDone) {
						return fmt.Errorf("err should be sql.ErrConnDone got: %w", err), false
					}
					return nil, true
				},
			},
			object: nil,
		},
		{
			name:    "prepareEmailQuery no result",
			prepare: prepareEmailQuery,
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(emailQuery),
					nil,
					nil,
				),
				err: func(err error) (error, bool) {
					if !errs.IsNotFound(err) {
						return fmt.Errorf("err should be zitadel.NotFoundError got: %w", err), false
					}
					return nil, true
				},
			},
			object: (*Email)(nil),
		},
		{
			name:    "prepareEmailQuery human found",
			prepare: prepareEmailQuery,
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(emailQuery),
					emailCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						"resource_owner",
						uint64(20211108),
						//domain.UserStateActive,
						"id",
						"email",
						true,
					},
				),
			},
			object: &Email{
				ID:            "id",
				CreationDate:  testNow,
				ChangeDate:    testNow,
				ResourceOwner: "resource_owner",
				Sequence:      20211108,
				//State:              domain.UserStateActive,
				Email:      "email",
				IsVerified: true,
			},
		},
		{
			name:    "prepareEmailQuery not human found (error)",
			prepare: prepareEmailQuery,
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(emailQuery),
					emailCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						"resource_owner",
						uint64(20211108),
						//domain.UserStateActive,
						nil,
						nil,
						nil,
					},
				),
				err: func(err error) (error, bool) {
					if !errs.IsPreconditionFailed(err) {
						return fmt.Errorf("err should be zitadel.PredconditionError got: %w", err), false
					}
					return nil, true
				},
			},
			object: (*Email)(nil),
		},
		{
			name:    "prepareEmailQuery sql err",
			prepare: prepareEmailQuery,
			want: want{
				sqlExpectations: mockQueryErr(
					regexp.QuoteMeta(emailQuery),
					sql.ErrConnDone,
				),
				err: func(err error) (error, bool) {
					if !errors.Is(err, sql.ErrConnDone) {
						return fmt.Errorf("err should be sql.ErrConnDone got: %w", err), false
					}
					return nil, true
				},
			},
			object: nil,
		},
		{
			name:    "preparePhoneQuery no result",
			prepare: preparePhoneQuery,
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(phoneQuery),
					nil,
					nil,
				),
				err: func(err error) (error, bool) {
					if !errs.IsNotFound(err) {
						return fmt.Errorf("err should be zitadel.NotFoundError got: %w", err), false
					}
					return nil, true
				},
			},
			object: (*Phone)(nil),
		},
		{
			name:    "preparePhoneQuery human found",
			prepare: preparePhoneQuery,
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(phoneQuery),
					phoneCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						"resource_owner",
						uint64(20211108),
						//domain.UserStateActive,
						"id",
						"phone",
						true,
					},
				),
			},
			object: &Phone{
				ID:            "id",
				CreationDate:  testNow,
				ChangeDate:    testNow,
				ResourceOwner: "resource_owner",
				Sequence:      20211108,
				//State:              domain.UserStateActive,
				Phone:      "phone",
				IsVerified: true,
			},
		},
		{
			name:    "preparePhoneQuery not human found (error)",
			prepare: preparePhoneQuery,
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(phoneQuery),
					phoneCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						"resource_owner",
						uint64(20211108),
						//domain.UserStateActive,
						nil,
						nil,
						nil,
					},
				),
				err: func(err error) (error, bool) {
					if !errs.IsPreconditionFailed(err) {
						return fmt.Errorf("err should be zitadel.PredconditionError got: %w", err), false
					}
					return nil, true
				},
			},
			object: (*Phone)(nil),
		},
		{
			name:    "preparePhoneQuery sql err",
			prepare: preparePhoneQuery,
			want: want{
				sqlExpectations: mockQueryErr(
					regexp.QuoteMeta(phoneQuery),
					sql.ErrConnDone,
				),
				err: func(err error) (error, bool) {
					if !errors.Is(err, sql.ErrConnDone) {
						return fmt.Errorf("err should be sql.ErrConnDone got: %w", err), false
					}
					return nil, true
				},
			},
			object: nil,
		},
		{
			name:    "prepareUserUniqueQuery no result",
			prepare: prepareUserUniqueQuery,
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(userUniqueQuery),
					nil,
					nil,
				),
				err: func(err error) (error, bool) {
					if !errs.IsNotFound(err) {
						return fmt.Errorf("err should be zitadel.NotFoundError got: %w", err), false
					}
					return nil, true
				},
			},
			object: true,
		},
		{
			name:    "prepareUserUniqueQuery found",
			prepare: prepareUserUniqueQuery,
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(userUniqueQuery),
					userUniqueCols,
					[]driver.Value{
						"id",
						domain.UserStateActive,
						"username",
						"id",
						"email",
						true,
					},
				),
			},
			object: false,
		},
		{
			name:    "prepareUserUniqueQuery sql err",
			prepare: prepareUserUniqueQuery,
			want: want{
				sqlExpectations: mockQueryErr(
					regexp.QuoteMeta(userUniqueQuery),
					sql.ErrConnDone,
				),
				err: func(err error) (error, bool) {
					if !errors.Is(err, sql.ErrConnDone) {
						return fmt.Errorf("err should be sql.ErrConnDone got: %w", err), false
					}
					return nil, true
				},
			},
			object: nil,
		},
		{
			name: "prepareNotifyUserQuery no result",
			prepare: func() (sq.SelectBuilder, func(*sql.Row) (*NotifyUser, error)) {
				return prepareNotifyUserQuery()
			},
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(notifyUserQuery),
					nil,
					nil,
				),
				err: func(err error) (error, bool) {
					if !errs.IsNotFound(err) {
						return fmt.Errorf("err should be zitadel.NotFoundError got: %w", err), false
					}
					return nil, true
				},
			},
			object: (*NotifyUser)(nil),
		},
		{
			name: "prepareNotifyUserQuery notify found",
			prepare: func() (sq.SelectBuilder, func(*sql.Row) (*NotifyUser, error)) {
				return prepareNotifyUserQuery()
			},
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(notifyUserQuery),
					notifyUserCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						"resource_owner",
						uint64(20211108),
						domain.UserStateActive,
						domain.UserTypeHuman,
						"username",
						database.StringArray{"login_name1", "login_name2"},
						"login_name1",
						//human
						"id",
						"first_name",
						"last_name",
						"nick_name",
						"display_name",
						"de",
						domain.GenderUnspecified,
						"avatar_key",
						//notify
						"id",
						"lastEmail",
						"verifiedEmail",
						"lastPhone",
						"verifiedPhone",
						true,
						1,
					},
				),
			},
			object: &NotifyUser{
				ID:                 "id",
				CreationDate:       testNow,
				ChangeDate:         testNow,
				ResourceOwner:      "resource_owner",
				Sequence:           20211108,
				State:              domain.UserStateActive,
				Type:               domain.UserTypeHuman,
				Username:           "username",
				LoginNames:         database.StringArray{"login_name1", "login_name2"},
				PreferredLoginName: "login_name1",
				FirstName:          "first_name",
				LastName:           "last_name",
				NickName:           "nick_name",
				DisplayName:        "display_name",
				AvatarKey:          "avatar_key",
				PreferredLanguage:  language.German,
				Gender:             domain.GenderUnspecified,
				LastEmail:          "lastEmail",
				VerifiedEmail:      "verifiedEmail",
				LastPhone:          "lastPhone",
				VerifiedPhone:      "verifiedPhone",
				PasswordSet:        true,
			},
		},
		{
			name: "prepareNotifyUserQuery not notify found (error)",
			prepare: func() (sq.SelectBuilder, func(*sql.Row) (*NotifyUser, error)) {
				return prepareNotifyUserQuery()
			},
			want: want{
				sqlExpectations: mockQuery(
					regexp.QuoteMeta(notifyUserQuery),
					notifyUserCols,
					[]driver.Value{
						"id",
						testNow,
						testNow,
						"resource_owner",
						uint64(20211108),
						domain.UserStateActive,
						domain.UserTypeHuman,
						"username",
						database.StringArray{"login_name1", "login_name2"},
						"login_name1",
						//human
						"id",
						"first_name",
						"last_name",
						"nick_name",
						"display_name",
						"de",
						domain.GenderUnspecified,
						"avatar_key",
						nil,
						nil,
						nil,
						nil,
						nil,
						nil,
						1,
					},
				),
				err: func(err error) (error, bool) {
					if !errs.IsPreconditionFailed(err) {
						return fmt.Errorf("err should be zitadel.PredconditionError got: %w", err), false
					}
					return nil, true
				},
			},
			object: (*NotifyUser)(nil),
		},
		{
			name: "prepareNotifyUserQuery sql err",
			prepare: func() (sq.SelectBuilder, func(*sql.Row) (*NotifyUser, error)) {
				return prepareNotifyUserQuery()
			},
			want: want{
				sqlExpectations: mockQueryErr(
					regexp.QuoteMeta(notifyUserQuery),
					sql.ErrConnDone,
				),
				err: func(err error) (error, bool) {
					if !errors.Is(err, sql.ErrConnDone) {
						return fmt.Errorf("err should be sql.ErrConnDone got: %w", err), false
					}
					return nil, true
				},
			},
			object: nil,
		},
		{
			name: "prepareUsersQuery no result",
			prepare: func() (sq.SelectBuilder, func(*sql.Rows) (*Users, error)) {
				return prepareUsersQuery()
			},
			want: want{
				sqlExpectations: mockQueries(
					regexp.QuoteMeta(usersQuery),
					nil,
					nil,
				),
				err: func(err error) (error, bool) {
					if !errs.IsNotFound(err) {
						return fmt.Errorf("err should be zitadel.NotFoundError got: %w", err), false
					}
					return nil, true
				},
			},
			object: &Users{Users: []*User{}},
		},
		{
			name: "prepareUsersQuery one result",
			prepare: func() (sq.SelectBuilder, func(*sql.Rows) (*Users, error)) {
				return prepareUsersQuery()
			},
			want: want{
				sqlExpectations: mockQueries(
					regexp.QuoteMeta(usersQuery),
					usersCols,
					[][]driver.Value{
						{
							"id",
							testNow,
							testNow,
							"resource_owner",
							uint64(20211108),
							domain.UserStateActive,
							domain.UserTypeHuman,
							"username",
							database.StringArray{"login_name1", "login_name2"},
							"login_name1",
							//human
							"id",
							"first_name",
							"last_name",
							"nick_name",
							"display_name",
							"de",
							domain.GenderUnspecified,
							"avatar_key",
							"email",
							true,
							"phone",
							true,
							//machine
							nil,
							nil,
							nil,
						},
					},
				),
			},
			object: &Users{
				SearchResponse: SearchResponse{
					Count: 1,
				},
				Users: []*User{
					{
						ID:                 "id",
						CreationDate:       testNow,
						ChangeDate:         testNow,
						ResourceOwner:      "resource_owner",
						Sequence:           20211108,
						State:              domain.UserStateActive,
						Type:               domain.UserTypeHuman,
						Username:           "username",
						LoginNames:         database.StringArray{"login_name1", "login_name2"},
						PreferredLoginName: "login_name1",
						Human: &Human{
							FirstName:         "first_name",
							LastName:          "last_name",
							NickName:          "nick_name",
							DisplayName:       "display_name",
							AvatarKey:         "avatar_key",
							PreferredLanguage: language.German,
							Gender:            domain.GenderUnspecified,
							Email:             "email",
							IsEmailVerified:   true,
							Phone:             "phone",
							IsPhoneVerified:   true,
						},
					},
				},
			},
		},
		{
			name: "prepareUsersQuery multiple results",
			prepare: func() (sq.SelectBuilder, func(*sql.Rows) (*Users, error)) {
				return prepareUsersQuery()
			},
			want: want{
				sqlExpectations: mockQueries(
					regexp.QuoteMeta(usersQuery),
					usersCols,
					[][]driver.Value{
						{
							"id",
							testNow,
							testNow,
							"resource_owner",
							uint64(20211108),
							domain.UserStateActive,
							domain.UserTypeHuman,
							"username",
							database.StringArray{"login_name1", "login_name2"},
							"login_name1",
							//human
							"id",
							"first_name",
							"last_name",
							"nick_name",
							"display_name",
							"de",
							domain.GenderUnspecified,
							"avatar_key",
							"email",
							true,
							"phone",
							true,
							//machine
							nil,
							nil,
							nil,
						},
						{
							"id",
							testNow,
							testNow,
							"resource_owner",
							uint64(20211108),
							domain.UserStateActive,
							domain.UserTypeMachine,
							"username",
							database.StringArray{"login_name1", "login_name2"},
							"login_name1",
							//human
							nil,
							nil,
							nil,
							nil,
							nil,
							nil,
							nil,
							nil,
							nil,
							nil,
							nil,
							nil,
							//machine
							"id",
							"name",
							"description",
						},
					},
				),
			},
			object: &Users{
				SearchResponse: SearchResponse{
					Count: 2,
				},
				Users: []*User{
					{
						ID:                 "id",
						CreationDate:       testNow,
						ChangeDate:         testNow,
						ResourceOwner:      "resource_owner",
						Sequence:           20211108,
						State:              domain.UserStateActive,
						Type:               domain.UserTypeHuman,
						Username:           "username",
						LoginNames:         database.StringArray{"login_name1", "login_name2"},
						PreferredLoginName: "login_name1",
						Human: &Human{
							FirstName:         "first_name",
							LastName:          "last_name",
							NickName:          "nick_name",
							DisplayName:       "display_name",
							AvatarKey:         "avatar_key",
							PreferredLanguage: language.German,
							Gender:            domain.GenderUnspecified,
							Email:             "email",
							IsEmailVerified:   true,
							Phone:             "phone",
							IsPhoneVerified:   true,
						},
					},
					{
						ID:                 "id",
						CreationDate:       testNow,
						ChangeDate:         testNow,
						ResourceOwner:      "resource_owner",
						Sequence:           20211108,
						State:              domain.UserStateActive,
						Type:               domain.UserTypeMachine,
						Username:           "username",
						LoginNames:         database.StringArray{"login_name1", "login_name2"},
						PreferredLoginName: "login_name1",
						Machine: &Machine{
							Name:        "name",
							Description: "description",
						},
					},
				},
			},
		},
		{
			name: "prepareUsersQuery sql err",
			prepare: func() (sq.SelectBuilder, func(*sql.Rows) (*Users, error)) {
				return prepareUsersQuery()
			},
			want: want{
				sqlExpectations: mockQueryErr(
					regexp.QuoteMeta(usersQuery),
					sql.ErrConnDone,
				),
				err: func(err error) (error, bool) {
					if !errors.Is(err, sql.ErrConnDone) {
						return fmt.Errorf("err should be sql.ErrConnDone got: %w", err), false
					}
					return nil, true
				},
			},
			object: nil,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertPrepare(t, tt.prepare, tt.object, tt.want.sqlExpectations, tt.want.err)
		})
	}
}
