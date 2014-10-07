package tokens

import (
	"fmt"
	"net/http"
	"testing"
	"time"

	"github.com/rackspace/gophercloud"
	"github.com/rackspace/gophercloud/openstack/identity/v2/tenants"
	th "github.com/rackspace/gophercloud/testhelper"
)

var expectedToken = &Token{
	ID:        "aaaabbbbccccdddd",
	ExpiresAt: time.Date(2014, time.January, 31, 15, 30, 58, 0, time.UTC),
	Tenant: tenants.Tenant{
		ID:          "fc394f2ab2df4114bde39905f800dc57",
		Name:        "test",
		Description: "There are many tenants. This one is yours.",
		Enabled:     true,
	},
}

var expectedServiceCatalog = &ServiceCatalog{
	Entries: []CatalogEntry{
		CatalogEntry{
			Name: "inscrutablewalrus",
			Type: "something",
			Endpoints: []Endpoint{
				Endpoint{
					PublicURL: "http://something0:1234/v2/",
					Region:    "region0",
				},
				Endpoint{
					PublicURL: "http://something1:1234/v2/",
					Region:    "region1",
				},
			},
		},
		CatalogEntry{
			Name: "arbitrarypenguin",
			Type: "else",
			Endpoints: []Endpoint{
				Endpoint{
					PublicURL: "http://else0:4321/v3/",
					Region:    "region0",
				},
			},
		},
	},
}

func tokenPost(t *testing.T, options gophercloud.AuthOptions, requestJSON string) CreateResult {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	client := gophercloud.ServiceClient{Endpoint: th.Endpoint()}

	th.Mux.HandleFunc("/tokens", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestHeader(t, r, "Accept", "application/json")
		th.TestJSONRequest(t, r, requestJSON)

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `
{
  "access": {
    "token": {
      "issued_at": "2014-01-30T15:30:58.000000Z",
      "expires": "2014-01-31T15:30:58Z",
      "id": "aaaabbbbccccdddd",
      "tenant": {
        "description": "There are many tenants. This one is yours.",
        "enabled": true,
        "id": "fc394f2ab2df4114bde39905f800dc57",
        "name": "test"
      }
    },
    "serviceCatalog": [
      {
        "endpoints": [
          {
            "publicURL": "http://something0:1234/v2/",
            "region": "region0"
          },
          {
            "publicURL": "http://something1:1234/v2/",
            "region": "region1"
          }
        ],
        "type": "something",
        "name": "inscrutablewalrus"
      },
      {
        "endpoints": [
          {
            "publicURL": "http://else0:4321/v3/",
            "region": "region0"
          }
        ],
        "type": "else",
        "name": "arbitrarypenguin"
      }
    ]
  }
}
    `)
	})

	return Create(&client, options)
}

func tokenPostErr(t *testing.T, options gophercloud.AuthOptions, expectedErr error) {
	th.SetupHTTP()
	defer th.TeardownHTTP()

	client := gophercloud.ServiceClient{Endpoint: th.Endpoint()}

	th.Mux.HandleFunc("/tokens", func(w http.ResponseWriter, r *http.Request) {
		th.TestMethod(t, r, "POST")
		th.TestHeader(t, r, "Content-Type", "application/json")
		th.TestHeader(t, r, "Accept", "application/json")

		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{}`)
	})

	actualErr := Create(&client, options).Err
	th.CheckEquals(t, expectedErr, actualErr)
}

func isSuccessful(t *testing.T, result CreateResult) {
	token, err := result.ExtractToken()
	th.AssertNoErr(t, err)
	th.CheckDeepEquals(t, expectedToken, token)

	serviceCatalog, err := result.ExtractServiceCatalog()
	th.AssertNoErr(t, err)
	th.CheckDeepEquals(t, expectedServiceCatalog, serviceCatalog)
}

func TestCreateWithPassword(t *testing.T) {
	options := gophercloud.AuthOptions{
		Username: "me",
		Password: "swordfish",
	}

	isSuccessful(t, tokenPost(t, options, `
    {
      "auth": {
        "passwordCredentials": {
          "username": "me",
          "password": "swordfish"
        }
      }
    }
  `))
}

func TestCreateTokenWithTenantID(t *testing.T) {
	options := gophercloud.AuthOptions{
		Username: "me",
		Password: "opensesame",
		TenantID: "fc394f2ab2df4114bde39905f800dc57",
	}

	isSuccessful(t, tokenPost(t, options, `
    {
      "auth": {
        "tenantId": "fc394f2ab2df4114bde39905f800dc57",
        "passwordCredentials": {
          "username": "me",
          "password": "opensesame"
        }
      }
    }
  `))
}

func TestCreateTokenWithTenantName(t *testing.T) {
	options := gophercloud.AuthOptions{
		Username:   "me",
		Password:   "opensesame",
		TenantName: "demo",
	}

	isSuccessful(t, tokenPost(t, options, `
    {
      "auth": {
        "tenantName": "demo",
        "passwordCredentials": {
          "username": "me",
          "password": "opensesame"
        }
      }
    }
  `))
}

func TestProhibitUserID(t *testing.T) {
	options := gophercloud.AuthOptions{
		Username: "me",
		UserID:   "1234",
		Password: "thing",
	}
	tokenPostErr(t, options, ErrUserIDProvided)
}

func TestProhibitAPIKey(t *testing.T) {
	options := gophercloud.AuthOptions{
		Username: "me",
		Password: "thing",
		APIKey:   "123412341234",
	}
	tokenPostErr(t, options, ErrAPIKeyProvided)
}

func TestProhibitDomainID(t *testing.T) {
	options := gophercloud.AuthOptions{
		Username: "me",
		Password: "thing",
		DomainID: "1234",
	}
	tokenPostErr(t, options, ErrDomainIDProvided)
}

func TestProhibitDomainName(t *testing.T) {
	options := gophercloud.AuthOptions{
		Username:   "me",
		Password:   "thing",
		DomainName: "wat",
	}
	tokenPostErr(t, options, ErrDomainNameProvided)
}

func TestRequireUsername(t *testing.T) {
	options := gophercloud.AuthOptions{
		Password: "thing",
	}
	tokenPostErr(t, options, ErrUsernameRequired)
}

func TestRequirePassword(t *testing.T) {
	options := gophercloud.AuthOptions{
		Username: "me",
	}
	tokenPostErr(t, options, ErrPasswordRequired)
}
