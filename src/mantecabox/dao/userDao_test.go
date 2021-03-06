package dao

import (
	"database/sql"
	"testing"

	"mantecabox/models"
	"mantecabox/utilities"

	"github.com/aodin/date"
	"github.com/gin-gonic/gin/json"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/require"
	"gopkg.in/guregu/null.v3"
)

const testUserInsert = `INSERT INTO users (email, password) VALUES ('testuser1', 'testpassword1');`

func TestUserPgDao_GetAll(t *testing.T) {
	testCases := []struct {
		name        string
		insertQuery string
		want        []models.User
	}{
		{
			"When the users table has some users, retrieve all them",
			`INSERT INTO users(email, password)
VALUES  ('testuser1', 'testpassword1'),
		('testuser2', 'testpassword2')`,
			[]models.User{
				{Credentials: models.Credentials{Email: "testuser1", Password: "testpassword1"}},
				{Credentials: models.Credentials{Email: "testuser2", Password: "testpassword2"}},
			},
		},
		{
			"When the users table is empty, retrieve an empty set",
			"",
			[]models.User{},
		},
		{
			"When the users table has some deleted users, don't retrieve them",
			`INSERT INTO users(deleted_at, email, password)
VALUES  (NULL, 'testuser1', 'testpassword1'),
		(NOW(), 'testuser2', 'testpassword2')`,
			[]models.User{
				{Credentials: models.Credentials{Email: "testuser1", Password: "testpassword1"}},
			},
		},
	}

	db := getDb(t)
	defer db.Close()

	for _, testCase := range testCases {
		cleanAndPopulateDb(db, testCase.insertQuery, t)

		t.Run(testCase.name, func(t *testing.T) {
			dao := UserPgDao{}
			got, err := dao.GetAll()
			require.NoError(t, err)

			// We ignore the timestamps as we don't need to get them compared
			// But we check they are valid (they were created today)
			for k, v := range got {
				createdAtDate := date.FromTime(v.CreatedAt.Time)
				updatedAtDate := date.FromTime(v.UpdatedAt.Time)
				require.True(t, createdAtDate.Within(date.SingleDay(createdAtDate)))
				require.True(t, updatedAtDate.Within(date.SingleDay(updatedAtDate)))
				got[k].CreatedAt = null.Time{}
				got[k].UpdatedAt = null.Time{}
			}
			require.Equal(t, testCase.want, got)
		})
	}
}

func TestUserPgDao_GetByPk(t *testing.T) {
	type args struct {
		email string
	}
	testCases := []struct {
		name        string
		insertQuery string
		args        args
		want        models.User
		wantErr     bool
	}{
		{
			"When you ask for an existent user, retrieve it",
			testUserInsert,
			args{email: "testuser1"},
			models.User{Credentials: models.Credentials{Email: "testuser1", Password: "testpassword1"}},
			false,
		},
		{
			"When you ask for an non-existent user, return an empty user and an error",
			"",
			args{email: "nonexistentuser"},
			models.User{},
			true,
		},
		{
			"When you ask for a user with an empty email, return an empty user and an error",
			"",
			args{},
			models.User{},
			true,
		},
		{
			"When you ask for a deleted user, return an empty user and an error",
			`INSERT INTO users (deleted_at, email, password) 
VALUES (NOW(), 'testuser1', 'testpassword1');`,
			args{email: "testuser1"},
			models.User{},
			true,
		},
	}

	db := getDb(t)
	defer db.Close()

	for _, testCase := range testCases {
		cleanAndPopulateDb(db, testCase.insertQuery, t)

		t.Run(testCase.name, func(t *testing.T) {
			dao := UserPgDao{}
			got, err := dao.GetByPk(testCase.args.email)
			requireUserEqualCheckingErrors(t, testCase.wantErr, err, testCase.want, got)
		})
	}
}

func TestUserPgDao_Create(t *testing.T) {
	user := models.User{Credentials: models.Credentials{Email: "testuser1", Password: "testpassword1"}}
	type args struct {
		user *models.User
	}
	testCases := []struct {
		name        string
		insertQuery string
		args        args
		want        models.User
		wantErr     bool
	}{
		{
			"When you create a new user, it gets inserted",
			"",
			args{user: &user},
			user,
			false,
		},
		{
			"When you create an already inserted user, return an empty user and an error",
			testUserInsert,
			args{user: &user},
			models.User{},
			true,
		},
		{
			"When you create a new user without email, return an empty user and an error",
			"",
			args{user: &models.User{Credentials: models.Credentials{Password: "testpassword1"}}},
			models.User{},
			true,
		},
		{
			"When you create a new user without password, return an empty user and an error",
			"",
			args{user: &models.User{Credentials: models.Credentials{Email: "testuser1"}}},
			models.User{},
			true,
		},
	}

	db := getDb(t)
	defer db.Close()

	for _, testCase := range testCases {
		cleanAndPopulateDb(db, testCase.insertQuery, t)

		t.Run(testCase.name, func(t *testing.T) {
			dao := UserPgDao{}
			got, err := dao.Create(testCase.args.user)
			requireUserEqualCheckingErrors(t, testCase.wantErr, err, testCase.want, got)
		})
	}
}
func TestUserPgDao_Update(t *testing.T) {
	user := models.User{Credentials: models.Credentials{Email: "testuser2", Password: "testpassword2"}}
	type args struct {
		email string
		user  *models.User
	}
	testCases := []struct {
		name        string
		insertQuery string
		args        args
		want        models.User
		wantErr     bool
	}{
		{
			"When you update an already inserted user, return the user updated",
			testUserInsert,
			args{email: "testuser1", user: &user},
			user,
			false,
		},
		{
			"When you update a non-existent user, return an empty user and an error",
			testUserInsert,
			args{email: "testuser2", user: &user},
			models.User{},
			true,
		},
		{
			"When you update a user with an empty email query, return an empty user and an error",
			testUserInsert,
			args{email: "", user: &user},
			models.User{},
			true,
		},
		{
			"When you update an inserted user without email, return an empty user and an error",
			testUserInsert,
			args{email: "testuser1", user: &models.User{Credentials: models.Credentials{Password: "testpassword2"}}},
			models.User{},
			true,
		},
		{
			"When you update an inserted user without password, return an empty user and an error",
			testUserInsert,
			args{email: "testuser1", user: &models.User{Credentials: models.Credentials{Email: "testuser2"}}},
			models.User{},
			true,
		},
	}

	db := getDb(t)
	defer db.Close()

	for _, testCase := range testCases {
		cleanAndPopulateDb(db, testCase.insertQuery, t)

		t.Run(testCase.name, func(t *testing.T) {
			dao := UserPgDao{}
			got, err := dao.Update(testCase.args.email, testCase.args.user)
			requireUserEqualCheckingErrors(t, testCase.wantErr, err, testCase.want, got)
		})
	}
}

func TestUserPgDao_Update2FA(t *testing.T) {
	db := getDb(t)
	defer db.Close()
	cleanAndPopulateDb(db, testUserInsert, t)
	updatedUser, err := UserPgDao{}.Update("testuser1", &models.User{
		Credentials: models.Credentials{
			Email:         "testuser1",
			Password:      "tespass",
			TwoFactorAuth: null.String{NullString: sql.NullString{Valid: true, String: "012345"}},
		},
	})
	require.NoError(t, err)
	require.True(t, updatedUser.TwoFactorTime.Valid)
	twoFactorTime := date.FromTime(updatedUser.TwoFactorTime.Time)
	require.True(t, twoFactorTime.Within(date.SingleDay(twoFactorTime)))
}

func TestUserPgDao_Delete(t *testing.T) {
	type args struct {
		email string
	}
	testCases := []struct {
		name        string
		insertQuery string
		args        args
		wantErr     bool
	}{
		{
			"When you delete an inserted user, return no error",
			testUserInsert,
			args{email: "testuser1"},
			false,
		},
		{
			"When you delete a non-existent user, return an error",
			testUserInsert,
			args{email: "testuser2"},
			true,
		},
		{
			"When you update a user with an empty email query, return an error",
			testUserInsert,
			args{email: ""},
			true,
		},
	}

	db := getDb(t)
	defer db.Close()

	for _, testCase := range testCases {
		cleanAndPopulateDb(db, testCase.insertQuery, t)

		t.Run(testCase.name, func(t *testing.T) {
			dao := UserPgDao{}
			err := dao.Delete(testCase.args.email)
			if testCase.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestJSONParsing(t *testing.T) {
	credentials := models.Credentials{}
	bytes, err := json.Marshal(credentials)
	require.NoError(t, err)
	require.Equal(t, `{"email":"","password":"","two_factor_auth":null,"two_factor_time":null}`, string(bytes))
}

func getDb(t *testing.T) *sql.DB {
	// Test preparation
	db, err := utilities.GetPgDb()
	if err != nil {
		logrus.Fatal("Unable to connnect with database: " + err.Error())
	}
	require.NotNil(t, db)
	require.NoError(t, err)
	return db
}

func cleanAndPopulateDb(db *sql.DB, insertQuery string, t *testing.T) {
	cleanDb(db)
	if insertQuery != "" {
		_, err := db.Exec(insertQuery)
		require.NoError(t, err)
	}
}

func cleanDb(db *sql.DB) {
	db.Exec("DELETE FROM users")
	db.Exec("DELETE FROM files")
	db.Exec("DELETE FROM login_attempts")
}

func requireUserEqualCheckingErrors(t *testing.T, wantErr bool, err error, expected models.User, actual models.User) {
	if wantErr {
		require.Error(t, err)
	} else {
		require.NoError(t, err)
		// We ignore the timestamps as we don't need to get them compared
		// But we check they are valid (they were created recently)
		createdAtDate := date.FromTime(actual.CreatedAt.Time)
		updatedAtDate := date.FromTime(actual.UpdatedAt.Time)
		require.True(t, createdAtDate.Within(date.SingleDay(createdAtDate)))
		require.True(t, updatedAtDate.Within(date.SingleDay(updatedAtDate)))
		actual.CreatedAt = null.Time{}
		actual.UpdatedAt = null.Time{}
	}
	require.Equal(t, expected, actual)
}
