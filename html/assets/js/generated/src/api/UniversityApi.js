/**
 * CTF Management API
 * API for managing CTF (Capture The Flag) games, teams, users, and services.
 *
 * The version of the OpenAPI document: 1.0.0
 * 
 *
 * NOTE: This class is auto generated by OpenAPI Generator (https://openapi-generator.tech).
 * https://openapi-generator.tech
 * Do not edit the class manually.
 *
 */


import ApiClient from "../ApiClient";
import UniversityResponse from '../model/UniversityResponse';

/**
* University service.
* @module api/UniversityApi
* @version 1.0.0
*/
export default class UniversityApi {

    /**
    * Constructs a new UniversityApi. 
    * @alias module:api/UniversityApi
    * @class
    * @param {module:ApiClient} [apiClient] Optional API client implementation to use,
    * default to {@link module:ApiClient#instance} if unspecified.
    */
    constructor(apiClient) {
        this.apiClient = apiClient || ApiClient.instance;
    }


    /**
     * Callback function to receive the result of the apiV1UniversitiesGet operation.
     * @callback module:api/UniversityApi~apiV1UniversitiesGetCallback
     * @param {String} error Error message, if any.
     * @param {Array.<module:model/UniversityResponse>} data The data returned by the service call.
     * @param {String} response The complete HTTP response.
     */

    /**
     * Retrieves a list of universities
     * This endpoint retrieves universities. It can optionally filter universities that match a specific term. 
     * @param {Object} opts Optional parameters
     * @param {String} [term] Optional search term to filter universities by name.
     * @param {module:api/UniversityApi~apiV1UniversitiesGetCallback} callback The callback function, accepting three arguments: error, data, response
     * data is of type: {@link Array.<module:model/UniversityResponse>}
     */
    apiV1UniversitiesGet(opts, callback) {
      opts = opts || {};
      let postBody = null;

      let pathParams = {
      };
      let queryParams = {
        'term': opts['term']
      };
      let headerParams = {
      };
      let formParams = {
      };

      let authNames = [];
      let contentTypes = [];
      let accepts = ['application/json'];
      let returnType = [UniversityResponse];
      return this.apiClient.callApi(
        '/api/v1/universities', 'GET',
        pathParams, queryParams, headerParams, formParams, postBody,
        authNames, contentTypes, accepts, returnType, null, callback
      );
    }


}
