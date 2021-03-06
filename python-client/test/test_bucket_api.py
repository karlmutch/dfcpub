# coding: utf-8

"""
    DFC

    DFC is a scalable object-storage based caching system with Amazon and Google Cloud backends.  # noqa: E501

    OpenAPI spec version: 1.1.0
    Contact: test@nvidia.com
    Generated by: https://openapi-generator.tech
"""


from __future__ import absolute_import

import unittest

import openapi_client
from openapi_client.api.bucket_api import BucketApi  # noqa: E501
from openapi_client.rest import ApiException

import os
import uuid
import logging as log

class TestBucketApi(unittest.TestCase):
    """BucketApi unit test stubs"""

    def setUp(self):
        configuration = openapi_client.Configuration()
        configuration.debug = False
        api_client = openapi_client.ApiClient(configuration)
        self.bucket = openapi_client.api.bucket_api.BucketApi(api_client)
        self.object = openapi_client.api.object_api.ObjectApi(api_client)
        self.models = openapi_client.models
        self.BUCKET_NAME = os.environ["BUCKET"]
        self.NEXT_TIER_URL = "http://foo.com"
        self.FILE_SIZE = 128
        self.SLEEP_LONG_SECONDS = 15
        self.created_objects = []
        self.created_buckets = []

    def tearDown(self):
        log.info("Cleaning up all created objects in cloud.")
        for object_name in self.created_objects:
            self.object.delete(self.BUCKET_NAME, object_name)
        log.info("Cleaning up all created local buckets.")
        input_params = self.models.InputParameters(
            self.models.Actions.DESTROYLB)
        for bucket_name in self.created_buckets:
            self.bucket.delete(bucket_name, input_params)

    def test_list_cloud_buckets(self):
        """
        Test case for list_names
        1.  Get list of all cloud and local buckets
        2.  Test that the specified bucket is present in one of cloud/local
            buckets.
        """
        log.info("GET BUCKET [%s]", self.BUCKET_NAME)
        bucket_names = self.bucket.list_names()
        self.assertTrue(len(bucket_names.cloud) > 0,
                        "Cloud bucket names are not present.")
        self.assertTrue(self.BUCKET_NAME in bucket_names.cloud
                        or self.BUCKET_NAME in bucket_names.local,
                        "Bucket name [%s] does not exist in cloud/local" %
                        self.BUCKET_NAME)

    def test_list_bucket(self):
        """
        1. Create bucket
        2. Create object in bucket
        3. List objects and verify that the object is present in the bucket
        :return:
        """
        bucket_name = self.__create_local_bucket()
        object_name, _ = self.__put_random_object(bucket_name)
        props = self.models.ObjectPropertyTypes.CHECKSUM
        requestParams = self.models.ObjectPropertiesRequestParams(props)
        input_params = self.models.InputParameters(
            self.models.Actions.LISTOBJECTS, value=requestParams)
        objectProperties = self.bucket.perform_operation(
            bucket_name, input_params)
        self.assertTrue(len(objectProperties.entries) == 1,
                        "Properties of object present in bucket not returned.")
        self.assertTrue(objectProperties.entries[0].name == object_name,
                        "Properties of object present in bucket not returned.")
        self.assertTrue(objectProperties.entries[0].checksum,
                        "Properties of object present in bucket not returned.")

    def test_bucket_get_create_delete(self):
        """
        1. Create bucket
        2. Get bucket
        3. Delete bucket
        4. Get bucket
        :return:
        """
        bucket_name = self.__create_local_bucket()
        self.assertTrue(self.__check_if_local_bucket_exists(bucket_name),
                        "Created bucket [%s] not in local buckets"
                        % bucket_name)
        input_params = self.models.InputParameters(
            self.models.Actions.DESTROYLB)
        log.info("DELETE BUCKET [%s]", bucket_name)
        self.bucket.delete(bucket_name, input_params)
        self.assertFalse(self.__check_if_local_bucket_exists(bucket_name),
                         "Deleted bucket [%s] in local buckets" % bucket_name)
        self.created_buckets.remove(bucket_name)

    @unittest.skip("This passes with 1 target and not with multiple since the buckets aren't synced yet")
    def test_rename_bucket(self):
        """
        1.  Create bucket
        2.  Rename bucket
        3.  Get old bucket
        4.  Get new bucket
        :return:
        """
        bucket_name = self.__create_local_bucket()
        new_bucket_name = uuid.uuid4().hex
        input_params = self.models.InputParameters(
            self.models.Actions.RENAMELB, new_bucket_name)
        log.info("RENAME BUCKET from [%s] to %s", bucket_name, new_bucket_name)
        self.bucket.perform_operation(bucket_name, input_params)
        self.assertFalse(self.__check_if_local_bucket_exists(bucket_name),
                         "Old bucket [%s] exists in local buckets"
                         % bucket_name)
        self.assertTrue(self.__check_if_local_bucket_exists(new_bucket_name),
                        "New bucket [%s] does not exist in local buckets"
                        % new_bucket_name)
        self.created_buckets.remove(bucket_name)
        self.created_buckets.append(new_bucket_name)

    def test_bucket_properties(self):
        """
        1. Set properties
        2. Get properties
        :return:
        """
        bucket_name = self.__create_local_bucket()
        input_params = self.models.InputParameters(self.models.Actions.SETPROPS)
        cksum_conf = self.models.BucketPropsCksum(
            checksum="inherit",
            validate_checksum_cold_get=False,
            validate_checksum_warm_get=False,
            enable_read_range_checksum=False,
        )
        input_params.value = self.models.BucketProps(
            self.models.CloudProvider.DFC,
            self.NEXT_TIER_URL,
            self.models.RWPolicy.NEXT_TIER,
            self.models.RWPolicy.NEXT_TIER,
            cksum_conf,
        )
        self.bucket.set_properties(bucket_name, input_params)

        headers = self.bucket.get_properties_with_http_info(bucket_name)[2]
        cloud_provider = headers[self.models.Headers.CLOUDPROVIDER]
        self.assertEqual(cloud_provider, self.models.CloudProvider.DFC,
                         "Incorrect CloudProvider in HEADER returned")
        versioning = headers[self.models.Headers.VERSIONING]
        self.assertEqual(versioning, self.models.Version.LOCAL,
                         "Incorrect Versioning in HEADER returned")
        next_tier_url = headers[self.models.Headers.NEXTTIERURL]
        self.assertEqual(next_tier_url, self.NEXT_TIER_URL,
                         "Incorrect NextTierURL in HEADER returned")
        read_policy = headers[self.models.Headers.READPOLICY]
        self.assertEqual(read_policy, self.models.RWPolicy.NEXT_TIER,
                         "Incorrect ReadPolicy in HEADER returned")
        write_policy = headers[self.models.Headers.WRITEPOLICY]
        self.assertEqual(write_policy, self.models.RWPolicy.NEXT_TIER,
                         "Incorrect WritePolicy in HEADER returned")

    def test_prefetch_list_objects(self):
        """
        1. Create object
        2. Evict object
        3. Prefetch object
        4. Check cached
        :return:
        """
        object_name, _ = self.__put_random_object()
        input_params = self.models.InputParameters(self.models.Actions.EVICT)
        log.info("Evict object list [%s/%s] InputParamaters [%s]",
                 self.BUCKET_NAME, object_name, input_params)
        self.object.delete(
            self.BUCKET_NAME, object_name, input_parameters=input_params)
        input_params.action = self.models.Actions.PREFETCH
        input_params.value = self.models.ListParameters(
            wait=True, objnames=[object_name])
        log.info("Prefetch object list [%s/%s] InputParamaters [%s]",
                 self.BUCKET_NAME, object_name, input_params)
        self.bucket.perform_operation(self.BUCKET_NAME, input_params)
        log.info("Get object [%s/%s] from cache",
                 self.BUCKET_NAME, object_name)
        self.object.get_properties(
            self.BUCKET_NAME, object_name, check_cached = True)

    def test_prefetch_range_objects(self):
        """
        1. Create object
        2. Evict object
        3. Prefetch object
        4. Check cached
        :return:
        """
        object_name, _ = self.__put_random_object()
        input_params = self.models.InputParameters(self.models.Actions.EVICT)
        log.info("Evict object list [%s/%s] InputParamaters [%s]",
                 self.BUCKET_NAME, object_name, input_params)
        self.object.delete(
            self.BUCKET_NAME, object_name, input_parameters=input_params)
        input_params.action = self.models.Actions.PREFETCH
        input_params.value = self.models.RangeParameters(wait=True,
                             prefix="", regex="", range="")
        log.info("Prefetch object range [%s/%s] InputParamaters [%s]",
                 self.BUCKET_NAME, object_name, input_params)
        self.bucket.perform_operation(self.BUCKET_NAME, input_params)
        log.info("Prefetch object list [%s/%s] InputParamaters [%s]",
                 self.BUCKET_NAME, object_name, input_params)
        self.object.get_properties(
            self.BUCKET_NAME, object_name, check_cached=True)

    def test_delete_list_objects(self):
        """
        1. Create object
        2. Delete object
        3. Get object
        :return:
        """
        object_name, _ = self.__put_random_object()
        input_params = self.models.InputParameters(
            self.models.Actions.DELETE, self.models.ListParameters(
            wait=True, objnames=[object_name]))
        log.info("Delete object list [%s/%s] InputParamaters [%s]",
                 self.BUCKET_NAME, object_name, input_params)
        self.object.delete(
            self.BUCKET_NAME, object_name, input_parameters=input_params)
        self.__execute_operation_on_unavailable_object(
            self.object.get, self.BUCKET_NAME, object_name)

    def test_delete_range_objects(self):
        """
        1. Create object
        2. Delete object
        3. Get object
        :return:
        """
        object_name, _ = self.__put_random_object()
        input_params = self.models.InputParameters(
            self.models.Actions.DELETE, self.models.RangeParameters(
                wait=True, prefix="", regex="", range=""))
        log.info("Delete object range [%s/%s] InputParamaters [%s]",
                 self.BUCKET_NAME, object_name, input_params)
        self.object.delete(
            self.BUCKET_NAME, object_name, input_parameters=input_params)
        self.__execute_operation_on_unavailable_object(
            self.object.get, self.BUCKET_NAME, object_name)

    def test_evict_list_objects(self):
        """
        1. Create object
        2. Evict object
        3. Check if object in cache
        :return:
        """
        object_name, _ = self.__put_random_object()
        input_params = self.models.InputParameters(
            self.models.Actions.EVICT, self.models.ListParameters(
            wait=True, objnames=[object_name]))
        log.info("Evict object list [%s/%s] InputParamaters [%s]",
                 self.BUCKET_NAME, object_name, input_params)
        self.object.delete(
            self.BUCKET_NAME, object_name, input_parameters=input_params)
        self.__execute_operation_on_unavailable_object(
            self.object.get_properties, self.BUCKET_NAME, object_name,
            check_cached=True)

    def test_evict_range_objects(self):
        """
        1. Create object
        2. Evict object
        3. Check if object in cache
        :return:
        """
        object_name, _ = self.__put_random_object()
        input_params = self.models.InputParameters(
            self.models.Actions.EVICT, self.models.RangeParameters(
                wait=True, prefix="", regex="", range=""))
        log.info("Evict object range [%s/%s] InputParamaters [%s]",
                 self.BUCKET_NAME, object_name, input_params)
        self.object.delete(
            self.BUCKET_NAME, object_name, input_parameters=input_params)
        self.__execute_operation_on_unavailable_object(
            self.object.get_properties, self.BUCKET_NAME, object_name,
            check_cached=True)

    def __check_if_local_bucket_exists(self, bucket_name):
        log.info("LIST BUCKET local names [%s]", bucket_name)
        bucket_names = self.bucket.list_names(loc=True)
        self.assertTrue(len(bucket_names.cloud) == 0,
                        "Cloud buckets returned when requesting for only "
                        "local buckets")
        return bucket_name in bucket_names.local

    def __put_random_object(self, bucket_name=None):
        bucket_name = bucket_name if bucket_name else self.BUCKET_NAME
        object_name = uuid.uuid4().hex
        input_object = os.urandom(self.FILE_SIZE)
        log.info("PUT object [%s/%s] size [%d]",
                 bucket_name, object_name, self.FILE_SIZE)
        self.object.put(bucket_name, object_name, body=input_object)
        if bucket_name == self.BUCKET_NAME:
            self.created_objects.append(object_name)
        return object_name, input_object

    def __create_local_bucket(self):
        bucket_name = uuid.uuid4().hex
        input_params = self.models.InputParameters(
            self.models.Actions.CREATELB)
        log.info("Create local bucket [%s]", bucket_name)
        self.bucket.perform_operation(bucket_name, input_params)
        self.created_buckets.append(bucket_name)
        return bucket_name

    def __execute_operation_on_unavailable_object(
            self, operation, bucket_name, object_name, **kwargs):
        log.info("[%s] on unavailable object [%s/%s]",
                 operation.__name__, bucket_name, object_name)
        with self.assertRaises(ApiException) as contextManager:
            operation(bucket_name, object_name, **kwargs)
        exception = contextManager.exception
        self.assertEqual(exception.status, 404)

if __name__ == '__main__':
    log.basicConfig(format='[%(levelname)s]:%(message)s', level=log.DEBUG)
    unittest.main()
