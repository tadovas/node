/*
 * Copyright (C) 2017 The "MysteriumNetwork/node" Authors.
 *
 * This program is free software: you can redistribute it and/or modify
 * it under the terms of the GNU General Public License as published by
 * the Free Software Foundation, either version 3 of the License, or
 * (at your option) any later version.
 *
 * This program is distributed in the hope that it will be useful,
 * but WITHOUT ANY WARRANTY; without even the implied warranty of
 * MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
 * GNU General Public License for more details.
 *
 * You should have received a copy of the GNU General Public License
 * along with this program.  If not, see <http://www.gnu.org/licenses/>.
 */

package registry

import (
	"github.com/ethereum/go-ethereum/common"
	"github.com/julienschmidt/httprouter"
	"github.com/mysterium/node/tequilapi/utils"
	"net/http"
)

// SignatureDTO represents Elliptic Curve signature parts
//
// swagger:model DecomposedSignatureDTO
type SignatureDTO struct {
	// S part of signature
	// example: "0x1321313212312..."
	R string
	// R part of signature
	// example: "0x1234563564354..."
	S string
	// Sign - 27 or 28 as expected by ethereum ecrecover function
	// example: 27
	V uint8
}

// PublicKeyPartsDTO represents ECDSA public key with first byte stripped (0x04) and splitted into two 32 bytes size arrays
//
// swagger:model PublicKeyPartsDTO
type PublicKeyPartsDTO struct {
	// First 32 bytes of public key in hex representation
	// example: "0x1321313212312..."
	Part1 string
	// Last 32 bytes of public key inx hex representation
	// example: "0x1321313212312..."
	Part2 string
}

// RegistrationDataDTO represents registration status and needed data for registering of given identity
//
// swagger:model RegistrationDataDTO
type RegistrationDataDTO struct {
	// Returns true if identity is registered in payments smart contract
	Registered bool

	PublicKey *PublicKeyPartsDTO `json:"PublicKey,omitempty"`

	Signature *SignatureDTO `json:"Signature,omitempty"`
}

type registrationEndpoint struct {
	dataProvider   RegistrationDataProvider
	statusProvider IdentityRegistry
}

func newRegistrationEndpoint(dataProvider RegistrationDataProvider, statusProvider IdentityRegistry) *registrationEndpoint {
	return &registrationEndpoint{
		dataProvider:   dataProvider,
		statusProvider: statusProvider,
	}
}

// swagger:operation GET /identities/{id}/registration Identity identityRegistration
// ---
// summary: Provide identity registration status
// description: Provides registration status for given identity, if identity is not registered - provides additional data required for identity registration
// parameters:
//   - in: path
//     name: id
//     description: hex address of identity
//     example: "0x0000000000000000000000000000000000000001"
//     type: string
// responses:
//   200:
//     description: Registration status and data
//     schema:
//       "$ref": "#/definitions/RegistrationDataDTO"
//   500:
//     description: Internal server error
//     schema:
//       "$ref": "#/definitions/ErrorMessageDTO"
func (endpoint *registrationEndpoint) RegistrationData(resp http.ResponseWriter, request *http.Request, params httprouter.Params) {
	id := params.ByName("id")

	identity := common.HexToAddress(id)

	isRegistered, err := endpoint.statusProvider.IsRegistered(identity)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	registrationResponse := RegistrationDataDTO{
		Registered: isRegistered,
	}

	if isRegistered {
		utils.WriteAsJSON(registrationResponse, resp)
		return
	}

	registrationData, err := endpoint.dataProvider.ProvideRegistrationData(identity)
	if err != nil {
		utils.SendError(resp, err, http.StatusInternalServerError)
		return
	}

	registrationResponse.PublicKey = &PublicKeyPartsDTO{
		Part1: common.ToHex(registrationData.PublicKey.Part1),
		Part2: common.ToHex(registrationData.PublicKey.Part2),
	}

	registrationResponse.Signature = &SignatureDTO{
		R: common.ToHex(registrationData.Signature.R[:]),
		S: common.ToHex(registrationData.Signature.S[:]),
		V: registrationData.Signature.V,
	}

	utils.WriteAsJSON(registrationResponse, resp)
}

// AddRegistrationEndpoint adds identity registration data endpoint to given http router
func AddRegistrationEndpoint(router *httprouter.Router, dataProvider RegistrationDataProvider, statusProvider IdentityRegistry) {

	registrationEndpoint := newRegistrationEndpoint(
		dataProvider,
		statusProvider,
	)

	router.GET("/identities/:id/registration", registrationEndpoint.RegistrationData)

}
