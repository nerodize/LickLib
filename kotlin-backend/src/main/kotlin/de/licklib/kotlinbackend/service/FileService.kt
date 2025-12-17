package de.licklib.kotlinbackend.service

import de.licklib.kotlinbackend.model.File
import de.licklib.kotlinbackend.repository.FileRepository
import org.springframework.stereotype.Service

@Service
class FileService(
    private val bucketService: BucketService,
    private val fileRepository: FileRepository
) {

    fun uploadFile(file: File): File {
        bucketService.uploadFile(
            file = file,
        )

        val uploadedFile = file

        val savedFile = fileRepository.save(uploadedFile)

        return savedFile
    }

}