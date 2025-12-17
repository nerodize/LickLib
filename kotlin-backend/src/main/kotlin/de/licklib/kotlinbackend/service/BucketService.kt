package de.licklib.kotlinbackend.service

import de.licklib.kotlinbackend.model.File
import de.licklib.kotlinbackend.properties.BucketProperties
import io.minio.MinioClient
import io.minio.PutObjectArgs
import org.springframework.stereotype.Service

@Service
class BucketService(
    private val minioClient: MinioClient,
    private val bucketProperties: BucketProperties
) {

//    fun getFile(fileName: String): File {
//       val fileInputStream = minioClient.getObject(GetObjectArgs
//            .builder()
//            .bucket(bucketProperties.name)
//            .`object`(fileName)
//            .build()
//        )
//    }

    fun uploadFile(file: File) {
        minioClient.putObject(
            PutObjectArgs.builder()
                .bucket(bucketProperties.name)
                .`object`(file.name)
                .stream(file.inputStream, file.size, -1)
                .contentType(file.contentType)
                .build()
        )
    }

}